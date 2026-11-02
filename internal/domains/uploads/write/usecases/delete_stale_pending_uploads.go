package usecases

import (
	"context"
	"errors"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/storage"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

// DeleteStalePendingUploadsResult is how many stale uploads the run removed.
type DeleteStalePendingUploadsResult struct {
	DeletedCount int
}

// DeleteStalePendingUploadsDependencies carries the upload store, the file storage and the event dispatcher.
type DeleteStalePendingUploadsDependencies struct {
	UploadWriteRepository repositories.UploadWriteRepository
	FileStorage           storage.FileStorage
	EventDispatcher       events.Dispatcher
}

// DeleteStalePendingUploads removes unattached uploads past models.PendingUploadRetention, files first.
// An upload that fails to delete is kept for the next run.
func DeleteStalePendingUploads(
	ctx context.Context,
	d DeleteStalePendingUploadsDependencies,
) (DeleteStalePendingUploadsResult, error) {
	staleUploads, err := d.UploadWriteRepository.ListUnusedPendingUploads(ctx)
	if err != nil {
		return DeleteStalePendingUploadsResult{}, err
	}

	deletedCount := 0
	var failures []error

	for _, upload := range staleUploads {
		err := deleteUploadWithFiles(ctx, deleteUploadWithFilesParams{
			UploadWriteRepository: d.UploadWriteRepository,
			FileStorage:           d.FileStorage,
			EventDispatcher:       d.EventDispatcher,
			Upload:                upload,
		})
		if err != nil {
			failures = append(failures, err)
			continue
		}

		deletedCount++
	}

	return DeleteStalePendingUploadsResult{DeletedCount: deletedCount}, errors.Join(failures...)
}
