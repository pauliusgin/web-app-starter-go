package usecases

import (
	"context"
	"errors"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/storage"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	uploadevents "github.com/pauliusgin/web_app_starter-be/internal/events/upload"
)

type deleteUploadWithFilesParams struct {
	UploadWriteRepository repositories.UploadWriteRepository
	FileStorage           storage.FileStorage
	EventDispatcher       events.Dispatcher
	Upload                models.Upload
}

func deleteUploadWithFiles(ctx context.Context, params deleteUploadWithFilesParams) error {
	for _, path := range params.Upload.StoredPaths() {
		if err := deleteStoredFileIfPresent(ctx, deleteStoredFileIfPresentParams{FileStorage: params.FileStorage, Path: path}); err != nil {
			return err
		}
	}

	if err := params.UploadWriteRepository.DeleteUploadByID(ctx, params.Upload.ID); err != nil {
		return err
	}

	uploadevents.DispatchUploadDeleted(ctx, params.EventDispatcher, uploadevents.NewUploadDeletedParams{
		UploadID: params.Upload.ID,
		UserID:   params.Upload.UserID,
		Purpose:  string(params.Upload.Purpose),
	})

	return nil
}

type deleteStoredFileIfPresentParams struct {
	FileStorage storage.FileStorage
	Path        string
}

func deleteStoredFileIfPresent(ctx context.Context, params deleteStoredFileIfPresentParams) error {
	err := params.FileStorage.DeleteFile(ctx, params.Path)
	if err == nil {
		return nil
	}

	if _, missing := errors.AsType[e.ErrStoredFileNotFound](err); missing {
		return nil
	}

	return e.ErrDeleteStoredFileFailed{Path: params.Path, Cause: err}
}
