// Package uploads holds the HTTP handlers for uploading files.
package uploads

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
	uploaderrors "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/usecases"
)

type uploadedFileResponse struct {
	ID          uuid.UUID `json:"id"`
	Purpose     string    `json:"purpose"`
	Kind        string    `json:"kind"`
	MimeType    string    `json:"mimeType"`
	OriginalURL string    `json:"originalUrl"`
	PreviewURL  *string   `json:"previewUrl"`
}

type uploadFilesResponse struct {
	Uploads []uploadedFileResponse `json:"uploads"`
}

const (
	// PurposeFormField is the multipart field naming what the files are for.
	PurposeFormField = "purpose"
	// FilesFormField is the multipart field each uploaded file is sent under.
	FilesFormField = "files"
)

const multipartMediaType = "multipart/form-data"

// MaxUploadRequestBytes stops a body no valid request could reach: the domain enforces the
// per-file limits, this only bounds the whole request (10 files of up to 5 MiB plus headroom).
const MaxUploadRequestBytes int64 = 55 << 20

// Parts over this stay on disk as temp files instead of in memory while the form is parsed.
const multipartMemoryBytes int64 = 32 << 20

// The server-wide timeouts suit JSON requests; receiving and storing ten large files needs longer.
const uploadFilesDeadline = 2 * time.Minute

// UploadFilesUseCase is the domain call the upload handler depends on.
type UploadFilesUseCase func(ctx context.Context, params usecases.UploadFilesParams) (usecases.UploadFilesResult, error)

// UploadFilesDependencies carries the collaborator the upload handler orchestrates.
type UploadFilesDependencies struct {
	UploadFiles UploadFilesUseCase
}

// UploadFiles returns a handler storing the multipart files the authenticated user sends.
func UploadFiles(d UploadFilesDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			slog.Error(handlers.RouteNotProtectedText, "path", r.URL.Path)
			handlers.WriteJSON(w, http.StatusInternalServerError,
				handlers.ErrorResponse{Code: handlers.ErrorCodeInternal, Message: unexpectedUploadFailureText})

			return
		}

		extendUploadDeadlines(w)

		form, err := readUploadForm(w, r)
		if err != nil {
			writeUploadFilesFailure(w, err)
			return
		}
		defer removeUploadForm(form)

		files, err := readIncomingFiles(form.File[FilesFormField])
		if err != nil {
			writeUploadFilesFailure(w, err)
			return
		}

		result, err := d.UploadFiles(r.Context(), usecases.UploadFilesParams{
			UserID:  userID,
			Purpose: models.Purpose(firstFormValue(form, PurposeFormField)),
			Files:   files,
		})
		if err != nil {
			writeUploadFilesFailure(w, err)
			return
		}

		handlers.WriteJSON(w, http.StatusCreated, buildUploadFilesResponse(result.Uploads))
	}
}

// extendUploadDeadlines is best effort: a writer that cannot move its deadlines keeps the server's.
func extendUploadDeadlines(w http.ResponseWriter) {
	controller := http.NewResponseController(w)
	deadline := time.Now().Add(uploadFilesDeadline)

	_ = controller.SetReadDeadline(deadline)
	_ = controller.SetWriteDeadline(deadline)
}

func readUploadForm(w http.ResponseWriter, r *http.Request) (*multipart.Form, error) {
	rawContentType := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(rawContentType)
	if err != nil || mediaType != multipartMediaType {
		return nil, ErrUploadContentTypeUnsupported{ContentType: rawContentType}
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadRequestBytes)

	if err := r.ParseMultipartForm(multipartMemoryBytes); err != nil {
		return nil, translateUploadFormFailure(err)
	}

	return r.MultipartForm, nil
}

func translateUploadFormFailure(err error) error {
	if bodyTooLarge, ok := errors.AsType[*http.MaxBytesError](err); ok {
		return handlers.ErrRequestBodyTooLarge{Limit: bodyTooLarge.Limit}
	}

	return ErrUploadFormUndecodable{Cause: err}
}

func removeUploadForm(form *multipart.Form) {
	if err := form.RemoveAll(); err != nil {
		slog.Warn(uploadFormCleanupFailedText, "error", err)
	}
}

func readIncomingFiles(fileHeaders []*multipart.FileHeader) ([]usecases.IncomingFile, error) {
	files := make([]usecases.IncomingFile, 0, len(fileHeaders))

	for _, fileHeader := range fileHeaders {
		content, err := readFileContent(fileHeader)
		if err != nil {
			return nil, err
		}

		files = append(files, usecases.IncomingFile{Name: fileHeader.Filename, Content: content})
	}

	return files, nil
}

func readFileContent(fileHeader *multipart.FileHeader) ([]byte, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, ErrUploadFormUndecodable{Cause: err}
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, ErrUploadFormUndecodable{Cause: err}
	}

	return content, nil
}

func firstFormValue(form *multipart.Form, field string) string {
	values := form.Value[field]
	if len(values) == 0 {
		return ""
	}

	return values[0]
}

func buildUploadFilesResponse(uploads []models.Upload) uploadFilesResponse {
	response := uploadFilesResponse{Uploads: make([]uploadedFileResponse, 0, len(uploads))}

	for _, upload := range uploads {
		response.Uploads = append(response.Uploads, uploadedFileResponse{
			ID:          upload.ID,
			Purpose:     string(upload.Purpose),
			Kind:        string(upload.Kind),
			MimeType:    upload.MimeType,
			OriginalURL: upload.OriginalURL,
			PreviewURL:  handlers.NullableString(upload.PreviewURL),
		})
	}

	return response
}

var uploadFormRules = []handlers.RejectionRule{
	handlers.RejectWhen[ErrUploadContentTypeUnsupported](
		handlers.Rejection{Status: http.StatusUnsupportedMediaType, Code: handlers.ErrorCodeUnsupportedContentType}),
	handlers.RejectWhen[ErrUploadFormUndecodable](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeRequestBodyInvalid}),
	handlers.RejectWhen[handlers.ErrRequestBodyTooLarge](
		handlers.Rejection{Status: http.StatusRequestEntityTooLarge, Code: handlers.ErrorCodeRequestBodyTooLarge}),
}

var uploadFilesRules = []handlers.RejectionRule{
	handlers.RejectWhen[uploaderrors.ErrUploadPurposeUnknown](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeUploadPurposeInvalid}),
	handlers.RejectWhen[uploaderrors.ErrUploadFilesMissing](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeUploadFilesMissing}),
	handlers.RejectWhen[uploaderrors.ErrTooManyUploadFiles](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeUploadTooManyFiles}),
	handlers.RejectWhen[uploaderrors.ErrUploadFileTypeUnsupported](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeUploadFileTypeUnsupported}),
	handlers.RejectWhen[uploaderrors.ErrUploadFileTypeNotAllowed](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeUploadFileTypeUnsupported}),
	handlers.RejectWhen[uploaderrors.ErrUploadFileTooLarge](
		handlers.Rejection{Status: http.StatusRequestEntityTooLarge, Code: handlers.ErrorCodeUploadFileTooLarge}),
	handlers.RejectWhen[uploaderrors.ErrImageDimensionsTooLarge](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeUploadImageDimensionsTooLarge}),
	handlers.RejectWhen[uploaderrors.ErrImageUnreadable](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeUploadFileUnreadable}),
}

func writeUploadFilesFailure(w http.ResponseWriter, err error) {
	handlers.WriteRejection(w, handlers.WriteRejectionParams{
		Err:                   err,
		RuleSets:              [][]handlers.RejectionRule{uploadFormRules, uploadFilesRules},
		UnexpectedFailureText: unexpectedUploadFailureText,
		RejectedLogText:       uploadRejectedText,
	})
}
