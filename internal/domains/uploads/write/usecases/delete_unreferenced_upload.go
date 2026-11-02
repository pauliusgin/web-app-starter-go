package usecases

import (
	"context"
	"errors"

	"github.com/google/uuid"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/storage"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

// DeleteUnreferencedUploadParams is the upload a resource stopped pointing at.
type DeleteUnreferencedUploadParams struct {
	UploadID uuid.UUID
}

// DeleteUnreferencedUploadDependencies carries the upload store, the file storage and the event dispatcher.
type DeleteUnreferencedUploadDependencies struct {
	UploadWriteRepository repositories.UploadWriteRepository
	FileStorage           storage.FileStorage
	EventDispatcher       events.Dispatcher
}

// DeleteUnreferencedUpload removes the upload and its files unless another resource still points at it.
// An upload that is still referenced or already gone is left alone without an error.
func DeleteUnreferencedUpload(
	ctx context.Context,
	d DeleteUnreferencedUploadDependencies,
	params DeleteUnreferencedUploadParams,
) error {
	upload, err := d.UploadWriteRepository.FindUnreferencedUploadByID(ctx, params.UploadID)
	if _, notFound := errors.AsType[e.ErrUploadNotFound](err); notFound {
		return nil
	}
	if err != nil {
		return err
	}

	return deleteUploadWithFiles(ctx, deleteUploadWithFilesParams{
		UploadWriteRepository: d.UploadWriteRepository,
		FileStorage:           d.FileStorage,
		EventDispatcher:       d.EventDispatcher,
		Upload:                upload,
	})
}
