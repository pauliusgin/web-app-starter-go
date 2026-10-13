package uploads

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/imaging"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/storage"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/usecases"
	usermodels "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

const uploadFilesPath = "/api/v1/uploads"

var uploadingUserID = uuid.MustParse("1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15")

var jpegContent = []byte("\xFF\xD8\xFF\xE0 jpeg bytes")

var pdfContent = []byte("%PDF-1.7\n pdf bytes")

type formFile struct {
	name    string
	content []byte
}

// uploadFilesHarness runs the real use case over in-memory ports, so a test sees what got stored.
type uploadFilesHarness struct {
	repository     *repositories.InMemoryUploadRepository
	fileStorage    *storage.InMemoryFileStorage
	imageProcessor *imaging.InMemoryImageProcessor
}

func newUploadFilesHarness(t *testing.T) uploadFilesHarness {
	t.Helper()

	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	return uploadFilesHarness{
		repository:     repositories.NewInMemoryUploadRepository(),
		fileStorage:    storage.NewInMemoryFileStorage(),
		imageProcessor: imaging.NewInMemoryImageProcessor(imaging.ImageInfo{Width: 800, Height: 600}),
	}
}

func (h uploadFilesHarness) handler() http.Handler {
	ignoreEvent := func(context.Context, events.NewEvent) error { return nil }
	dependencies := usecases.UploadFilesDependencies{
		UploadRepository: h.repository,
		FileStorage:      h.fileStorage,
		ImageProcessor:   h.imageProcessor,
		EventDispatcher:  events.Dispatcher{Recorder: ignoreEvent, Publisher: ignoreEvent},
		AppEnv:           "test",
	}

	return UploadFiles(UploadFilesDependencies{
		UploadFiles: func(ctx context.Context, params usecases.UploadFilesParams) (usecases.UploadFilesResult, error) {
			return usecases.UploadFiles(ctx, dependencies, params)
		},
	})
}

func authenticateForTest(userID uuid.UUID) func(http.Handler) http.Handler {
	return middleware.RequireAuthenticatedUser(middleware.RequireAuthenticatedUserDependencies{
		ParseAccessToken: func(string) (jwt.AccessTokenClaims, error) {
			return jwt.AccessTokenClaims{UserID: userID}, nil
		},
		FindUserByID: func(_ context.Context, id uuid.UUID) (usermodels.User, error) {
			return usermodels.User{ID: id, Email: "accountant@example.com"}, nil
		},
	})
}

func multipartBody(t *testing.T, purpose string, files ...formFile) (*bytes.Buffer, string) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if purpose != "" {
		if err := writer.WriteField(PurposeFormField, purpose); err != nil {
			t.Fatalf("writing the purpose field: %v", err)
		}
	}
	for _, file := range files {
		part, err := writer.CreateFormFile(FilesFormField, file.name)
		if err != nil {
			t.Fatalf("creating the %s part: %v", file.name, err)
		}
		if _, err := part.Write(file.content); err != nil {
			t.Fatalf("writing the %s part: %v", file.name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("closing the multipart body: %v", err)
	}

	return body, writer.FormDataContentType()
}

type postUploadFilesParams struct {
	body        io.Reader
	contentType string
	skipGuard   bool
}

func (h uploadFilesHarness) post(t *testing.T, params postUploadFilesParams) *http.Response {
	t.Helper()

	served := h.handler()
	if !params.skipGuard {
		served = authenticateForTest(uploadingUserID)(served)
	}

	request := httptest.NewRequest(http.MethodPost, uploadFilesPath, params.body)
	request.Header.Set("Authorization", "Bearer header.payload.signature")
	request.Header.Set("Content-Type", params.contentType)

	recorder := httptest.NewRecorder()
	served.ServeHTTP(recorder, request)

	response := recorder.Result()
	t.Cleanup(func() { response.Body.Close() })

	return response
}

func (h uploadFilesHarness) postFiles(t *testing.T, purpose string, files ...formFile) *http.Response {
	t.Helper()

	body, contentType := multipartBody(t, purpose, files...)

	return h.post(t, postUploadFilesParams{body: body, contentType: contentType})
}

func decodeUploadFilesResponse(t *testing.T, response *http.Response) uploadFilesResponse {
	t.Helper()

	decoded := uploadFilesResponse{}
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("decoding the response body: %v", err)
	}

	return decoded
}

func assertErrorResponse(t *testing.T, response *http.Response, wantStatus int, wantCode handlers.ErrorCode) {
	t.Helper()

	if response.StatusCode != wantStatus {
		t.Errorf("status = %d, want %d", response.StatusCode, wantStatus)
	}

	decoded := handlers.ErrorResponse{}
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("decoding the error body: %v", err)
	}
	if decoded.Code != wantCode {
		t.Errorf("code = %q, want %q (message %q)", decoded.Code, wantCode, decoded.Message)
	}
}

func TestUploadFilesAnswersCreatedWithBothURLsOfAnImage(t *testing.T) {
	harness := newUploadFilesHarness(t)

	response := harness.postFiles(t, "profile_picture", formFile{name: "me.jpg", content: jpegContent})

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusCreated)
	}

	decoded := decodeUploadFilesResponse(t, response)
	if len(decoded.Uploads) != 1 || len(harness.repository.Uploads) != 1 {
		t.Fatalf("answered %d uploads, saved %d, want 1 each", len(decoded.Uploads), len(harness.repository.Uploads))
	}

	saved := harness.repository.Uploads[0]
	answered := decoded.Uploads[0]
	if answered.PreviewURL == nil || *answered.PreviewURL != saved.PreviewURL || saved.PreviewURL == "" {
		t.Errorf("previewUrl = %v, want the saved preview URL %q", answered.PreviewURL, saved.PreviewURL)
	}

	answered.PreviewURL = nil
	want := uploadedFileResponse{
		ID:          saved.ID,
		Purpose:     "profile_picture",
		Kind:        "image",
		MimeType:    "image/jpeg",
		OriginalURL: saved.OriginalURL,
	}
	if answered != want {
		t.Errorf("upload = %+v, want %+v", answered, want)
	}
	if saved.UserID != uploadingUserID {
		t.Errorf("owner = %s, want the authenticated user %s", saved.UserID, uploadingUserID)
	}
	if want.OriginalURL == "" {
		t.Error("originalUrl is empty, want it set")
	}
}

func TestUploadFilesAnswersANullPreviewURLForADocument(t *testing.T) {
	harness := newUploadFilesHarness(t)

	response := harness.postFiles(t, "document_attachment", formFile{name: "document.pdf", content: pdfContent})

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusCreated)
	}

	raw := map[string][]map[string]any{}
	if err := json.NewDecoder(response.Body).Decode(&raw); err != nil {
		t.Fatalf("decoding the response body: %v", err)
	}
	previewURL, present := raw["uploads"][0]["previewUrl"]
	if !present || previewURL != nil {
		t.Errorf("upload = %v, want previewUrl present as null for a document", raw["uploads"][0])
	}
}

func TestUploadFilesRejectsWhatTheDomainRefuses(t *testing.T) {
	cases := []struct {
		name       string
		purpose    string
		files      []formFile
		wantStatus int
		wantCode   handlers.ErrorCode
	}{
		{
			name:       "missing purpose",
			files:      []formFile{{name: "me.jpg", content: jpegContent}},
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodeUploadPurposeInvalid,
		},
		{
			name:       "no files",
			purpose:    "profile_picture",
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodeUploadFilesMissing,
		},
		{
			name:       "unsupported format",
			purpose:    "document_attachment",
			files:      []formFile{{name: "notes.txt", content: []byte("plain notes")}},
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodeUploadFileTypeUnsupported,
		},
		{
			name:       "document as profile picture",
			purpose:    "profile_picture",
			files:      []formFile{{name: "cv.pdf", content: pdfContent}},
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodeUploadFileTypeUnsupported,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newUploadFilesHarness(t)

			response := harness.postFiles(t, testCase.purpose, testCase.files...)

			assertErrorResponse(t, response, testCase.wantStatus, testCase.wantCode)
			if len(harness.fileStorage.Files) != 0 || len(harness.repository.Uploads) != 0 {
				t.Errorf("stored %d files and %d uploads, want none",
					len(harness.fileStorage.Files), len(harness.repository.Uploads))
			}
		})
	}
}

func TestUploadFilesRejectsImagesTheProcessorRefuses(t *testing.T) {
	cases := []struct {
		name     string
		arrange  func(processor *imaging.InMemoryImageProcessor)
		wantCode handlers.ErrorCode
	}{
		{
			name: "too wide",
			arrange: func(processor *imaging.InMemoryImageProcessor) {
				processor.Dimensions = imaging.ImageInfo{Width: models.MaxImageEdgePixels + 1, Height: 10}
			},
			wantCode: handlers.ErrorCodeUploadImageDimensionsTooLarge,
		},
		{
			name: "undecodable",
			arrange: func(processor *imaging.InMemoryImageProcessor) {
				processor.InspectFailure = errors.New("corrupt jpeg")
			},
			wantCode: handlers.ErrorCodeUploadFileUnreadable,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newUploadFilesHarness(t)
			testCase.arrange(harness.imageProcessor)

			response := harness.postFiles(t, "profile_picture", formFile{name: "me.jpg", content: jpegContent})

			assertErrorResponse(t, response, http.StatusBadRequest, testCase.wantCode)
		})
	}
}

func TestUploadFilesRejectsABodyThatIsNotAMultipartForm(t *testing.T) {
	cases := []struct {
		name        string
		body        string
		contentType string
		wantStatus  int
		wantCode    handlers.ErrorCode
	}{
		{
			name:        "json body",
			body:        `{"purpose":"profile_picture"}`,
			contentType: "application/json",
			wantStatus:  http.StatusUnsupportedMediaType,
			wantCode:    handlers.ErrorCodeUnsupportedContentType,
		},
		{
			name:        "truncated multipart body",
			body:        "--boundary\r\nContent-Disposition: form-data; name=\"purpose\"\r\n\r\nprofile",
			contentType: "multipart/form-data; boundary=boundary",
			wantStatus:  http.StatusBadRequest,
			wantCode:    handlers.ErrorCodeRequestBodyInvalid,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			harness := newUploadFilesHarness(t)

			response := harness.post(t, postUploadFilesParams{
				body:        strings.NewReader(testCase.body),
				contentType: testCase.contentType,
			})

			assertErrorResponse(t, response, testCase.wantStatus, testCase.wantCode)
		})
	}
}

func TestTranslateUploadFormFailureReportsAnOversizedBody(t *testing.T) {
	translated := translateUploadFormFailure(&http.MaxBytesError{Limit: MaxUploadRequestBytes})

	tooLarge, ok := errors.AsType[handlers.ErrRequestBodyTooLarge](translated)
	if !ok || tooLarge.Limit != MaxUploadRequestBytes {
		t.Fatalf("translated = %v, want ErrRequestBodyTooLarge carrying the limit", translated)
	}

	rejection, matched := handlers.SelectRejection(translated, uploadFormRules)
	if !matched || rejection.Status != http.StatusRequestEntityTooLarge {
		t.Errorf("rejection = %+v, want 413", rejection)
	}
}

func TestUploadFilesAnswersInternalErrorWhenStorageFails(t *testing.T) {
	harness := newUploadFilesHarness(t)
	harness.fileStorage.StoreFailure = errors.New("bucket unreachable")

	response := harness.postFiles(t, "profile_picture", formFile{name: "me.jpg", content: jpegContent})

	assertErrorResponse(t, response, http.StatusInternalServerError, handlers.ErrorCodeInternal)
}

func TestUploadFilesAnswersInternalErrorWhenTheRouteIsUnprotected(t *testing.T) {
	harness := newUploadFilesHarness(t)
	body, contentType := multipartBody(t, "profile_picture", formFile{name: "me.jpg", content: jpegContent})

	response := harness.post(t, postUploadFilesParams{body: body, contentType: contentType, skipGuard: true})

	assertErrorResponse(t, response, http.StatusInternalServerError, handlers.ErrorCodeInternal)
	if len(harness.repository.Uploads) != 0 {
		t.Errorf("saved uploads = %d, want none", len(harness.repository.Uploads))
	}
}
