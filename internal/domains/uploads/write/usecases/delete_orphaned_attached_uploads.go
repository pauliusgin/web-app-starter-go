package usecases

import (
	"context"
	"errors"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/storage"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

// DeleteOrphanedAttachedUploadsResult is how many orphaned uploads the run removed.
type DeleteOrphanedAttachedUploadsResult struct {
	DeletedCount int
}

// DeleteOrphanedAttachedUploadsDependencies carries the upload store, the file storage and the event dispatcher.
type DeleteOrphanedAttachedUploadsDependencies struct {
	UploadWriteRepository repositories.UploadWriteRepository
	FileStorage           storage.FileStorage
	EventDispatcher       events.Dispatcher
}

// DeleteOrphanedAttachedUploads removes attached uploads no resource points at any more, files first.
// An upload that fails to delete is kept for the next run.
func DeleteOrphanedAttachedUploads(
	ctx context.Context,
	d DeleteOrphanedAttachedUploadsDependencies,
) (DeleteOrphanedAttachedUploadsResult, error) {
	orphanedUploads, err := d.UploadWriteRepository.ListOrphanedAttachedUploads(ctx)
	if err != nil {
		return DeleteOrphanedAttachedUploadsResult{}, err
	}

	deletedCount := 0
	var failures []error

	for _, upload := range orphanedUploads {
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

	return DeleteOrphanedAttachedUploadsResult{DeletedCount: deletedCount}, errors.Join(failures...)
}
