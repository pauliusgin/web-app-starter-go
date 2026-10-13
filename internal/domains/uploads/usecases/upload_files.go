// Package usecases holds the upload use cases: storing users' files and purging unattached ones.
package usecases

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/imaging"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/storage"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	uploadevents "github.com/pauliusgin/web_app_starter-be/internal/events/upload"
)

// IncomingFile is one file of an upload request, as the client sent it.
type IncomingFile struct {
	Name    string
	Content []byte
}

// UploadFilesParams is who uploads, what the files are for, and the files themselves.
type UploadFilesParams struct {
	UserID  uuid.UUID
	Purpose models.Purpose
	Files   []IncomingFile
}

// UploadFilesResult is every upload stored and recorded as pending, in the order the files came.
type UploadFilesResult struct {
	Uploads []models.Upload
}

// UploadFilesDependencies carries the collaborators storing and recording uploads needs.
type UploadFilesDependencies struct {
	UploadRepository repositories.UploadRepository
	FileStorage      storage.FileStorage
	ImageProcessor   imaging.ImageProcessor
	EventDispatcher  events.Dispatcher
	AppEnv           string
}

type validatedFile struct {
	Name    string
	Content []byte
	Format  models.FileFormat
	Image   imaging.ImageInfo
}

// UploadFiles validates every file, then stores each with its image preview and records it as
// pending. Nothing is stored unless every file passes validation.
func UploadFiles(
	ctx context.Context,
	d UploadFilesDependencies,
	params UploadFilesParams,
) (UploadFilesResult, error) {
	rules, err := checkUploadRequest(params)
	if err != nil {
		return UploadFilesResult{}, err
	}

	validatedFiles, err := validateIncomingFiles(ctx, validateIncomingFilesParams{
		ImageProcessor: d.ImageProcessor,
		Rules:          rules,
		Purpose:        params.Purpose,
		Files:          params.Files,
	})
	if err != nil {
		return UploadFilesResult{}, err
	}

	uploads := make([]models.Upload, 0, len(validatedFiles))
	for _, file := range validatedFiles {
		upload, err := storeUpload(ctx, d, storeUploadParams{
			UploadID: uuid.New(),
			UserID:   params.UserID,
			Purpose:  params.Purpose,
			Preview:  rules.Preview,
			File:     file,
		})
		if err != nil {
			return UploadFilesResult{}, err
		}

		uploads = append(uploads, upload)
	}

	return UploadFilesResult{Uploads: uploads}, nil
}

func checkUploadRequest(params UploadFilesParams) (models.PurposeRules, error) {
	rules, known := models.RulesForPurpose(params.Purpose)
	if !known {
		return models.PurposeRules{}, e.ErrUploadPurposeUnknown{Purpose: string(params.Purpose)}
	}
	if len(params.Files) == 0 {
		return models.PurposeRules{}, e.ErrUploadFilesMissing{}
	}
	if len(params.Files) > rules.MaxFiles {
		return models.PurposeRules{}, e.ErrTooManyUploadFiles{
			Count: len(params.Files),
			Limit: rules.MaxFiles,
		}
	}

	return rules, nil
}

type validateIncomingFilesParams struct {
	ImageProcessor imaging.ImageProcessor
	Rules          models.PurposeRules
	Purpose        models.Purpose
	Files          []IncomingFile
}

func validateIncomingFiles(ctx context.Context, params validateIncomingFilesParams) ([]validatedFile, error) {
	validatedFiles := make([]validatedFile, 0, len(params.Files))

	for _, file := range params.Files {
		validated, err := validateIncomingFile(ctx, validateIncomingFileParams{
			ImageProcessor: params.ImageProcessor,
			Rules:          params.Rules,
			Purpose:        params.Purpose,
			File:           file,
		})
		if err != nil {
			return nil, err
		}

		validatedFiles = append(validatedFiles, validated)
	}

	return validatedFiles, nil
}

type validateIncomingFileParams struct {
	ImageProcessor imaging.ImageProcessor
	Rules          models.PurposeRules
	Purpose        models.Purpose
	File           IncomingFile
}

func validateIncomingFile(ctx context.Context, params validateIncomingFileParams) (validatedFile, error) {
	file := params.File

	format, detectedMimeType, accepted := models.DetectFileFormat(file.Content)
	if !accepted {
		return validatedFile{}, e.ErrUploadFileTypeUnsupported{FileName: file.Name, DetectedMimeType: detectedMimeType}
	}
	if !params.Rules.AllowsKind(format.Kind) {
		return validatedFile{}, e.ErrUploadFileTypeNotAllowed{
			FileName: file.Name,
			Kind:     string(format.Kind),
			Purpose:  string(params.Purpose),
		}
	}

	sizeBytes := int64(len(file.Content))
	if sizeBytes > format.MaxSizeBytes {
		return validatedFile{}, e.ErrUploadFileTooLarge{
			FileName:     file.Name,
			SizeBytes:    sizeBytes,
			MaxSizeBytes: format.MaxSizeBytes,
		}
	}

	validated := validatedFile{Name: file.Name, Content: file.Content, Format: format}
	if format.Kind != models.KindImage {
		return validated, nil
	}

	image, err := inspectUploadedImage(ctx, params.ImageProcessor, file)
	if err != nil {
		return validatedFile{}, err
	}
	validated.Image = image

	return validated, nil
}

func inspectUploadedImage(
	ctx context.Context,
	imageProcessor imaging.ImageProcessor,
	file IncomingFile,
) (imaging.ImageInfo, error) {
	image, err := imageProcessor.InspectImage(ctx, file.Content)
	if err != nil {
		return imaging.ImageInfo{}, e.ErrImageUnreadable{FileName: file.Name, Cause: err}
	}

	if image.Width > models.MaxImageEdgePixels || image.Height > models.MaxImageEdgePixels {
		return imaging.ImageInfo{}, e.ErrImageDimensionsTooLarge{
			FileName:      file.Name,
			Width:         image.Width,
			Height:        image.Height,
			MaxEdgePixels: models.MaxImageEdgePixels,
		}
	}

	return image, nil
}

type storeUploadParams struct {
	UploadID uuid.UUID
	UserID   uuid.UUID
	Purpose  models.Purpose
	Preview  models.PreviewSpec
	File     validatedFile
}

// storeUpload stores one file and records it. A failure discards whatever this file already stored;
// uploads recorded for earlier files stay pending and are swept by DeleteStalePendingUploads.
func storeUpload(ctx context.Context, d UploadFilesDependencies, params storeUploadParams) (models.Upload, error) {
	newUpload, storedPaths, err := storeUploadFiles(ctx, d, params)
	if err != nil {
		discardStoredFiles(ctx, d.FileStorage, storedPaths)
		return models.Upload{}, err
	}

	saved, err := d.UploadRepository.SaveUpload(ctx, newUpload)
	if err != nil {
		discardStoredFiles(ctx, d.FileStorage, storedPaths)
		return models.Upload{}, err
	}

	uploadevents.DispatchUploadCreated(ctx, d.EventDispatcher, uploadevents.NewUploadCreatedParams{
		UploadID: saved.ID,
		UserID:   saved.UserID,
		Purpose:  string(saved.Purpose),
		Kind:     string(saved.Kind),
	})

	return saved, nil
}

// storeUploadFiles returns the paths it stored even on failure, so the caller can discard them.
func storeUploadFiles(
	ctx context.Context,
	d UploadFilesDependencies,
	params storeUploadParams,
) (models.NewUpload, []string, error) {
	file := params.File
	paths := models.BuildStoragePaths(models.StoragePathsParams{
		AppEnv:    d.AppEnv,
		UserID:    params.UserID,
		Purpose:   params.Purpose,
		UploadID:  params.UploadID,
		Extension: file.Format.Extension,
	})

	original, err := storeFileAt(ctx, d.FileStorage, storage.StoreFileParams{
		Path:        paths.Original,
		ContentType: file.Format.MimeType,
		Content:     file.Content,
	})
	if err != nil {
		return models.NewUpload{}, nil, err
	}

	newUpload := models.NewUpload{
		ID:           params.UploadID,
		UserID:       params.UserID,
		Purpose:      params.Purpose,
		Kind:         file.Format.Kind,
		MimeType:     file.Format.MimeType,
		SizeBytes:    int64(len(file.Content)),
		Width:        file.Image.Width,
		Height:       file.Image.Height,
		OriginalPath: paths.Original,
		OriginalURL:  original.URL,
	}
	storedPaths := []string{paths.Original}

	if file.Format.Kind != models.KindImage {
		return newUpload, storedPaths, nil
	}

	preview, err := storeImagePreview(ctx, d, storeImagePreviewParams{
		Path:    paths.Preview,
		Preview: params.Preview,
		File:    file,
	})
	if err != nil {
		return models.NewUpload{}, storedPaths, err
	}

	newUpload.PreviewPath = paths.Preview
	newUpload.PreviewURL = preview.URL

	return newUpload, append(storedPaths, paths.Preview), nil
}

type storeImagePreviewParams struct {
	Path    string
	Preview models.PreviewSpec
	File    validatedFile
}

func storeImagePreview(
	ctx context.Context,
	d UploadFilesDependencies,
	params storeImagePreviewParams,
) (storage.StoredFile, error) {
	previewContent, err := d.ImageProcessor.CreatePreview(ctx, imaging.CreatePreviewParams{
		Content:    params.File.Content,
		EdgePixels: params.Preview.EdgePixels,
		Crop:       params.Preview.Crop,
	})
	if err != nil {
		return storage.StoredFile{}, e.ErrCreatePreviewFailed{FileName: params.File.Name, Cause: err}
	}

	return storeFileAt(ctx, d.FileStorage, storage.StoreFileParams{
		Path:        params.Path,
		ContentType: models.PreviewMimeType,
		Content:     previewContent,
	})
}

func storeFileAt(
	ctx context.Context,
	fileStorage storage.FileStorage,
	params storage.StoreFileParams,
) (storage.StoredFile, error) {
	stored, err := fileStorage.StoreFile(ctx, params)
	if err != nil {
		return storage.StoredFile{}, e.ErrStoreFileFailed{Path: params.Path, Cause: err}
	}

	return stored, nil
}

// discardStoredFiles is best effort: a file it cannot remove is logged and left orphaned in storage.
func discardStoredFiles(ctx context.Context, fileStorage storage.FileStorage, paths []string) {
	for _, path := range paths {
		if err := fileStorage.DeleteFile(ctx, path); err != nil {
			slog.Error(e.DiscardStoredFileFailedText, "path", path, "error", err)
		}
	}
}
