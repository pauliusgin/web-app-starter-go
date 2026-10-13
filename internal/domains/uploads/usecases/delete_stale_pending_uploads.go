package usecases

import (
	"context"
	"errors"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/storage"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	uploadevents "github.com/pauliusgin/web_app_starter-be/internal/events/upload"
)

// DeleteStalePendingUploadsResult is how many stale uploads the run removed.
type DeleteStalePendingUploadsResult struct {
	DeletedCount int
}

// DeleteStalePendingUploadsDependencies carries the collaborators purging stale uploads needs.
type DeleteStalePendingUploadsDependencies struct {
	UploadRepository repositories.UploadRepository
	FileStorage      storage.FileStorage
	EventDispatcher  events.Dispatcher
}

// DeleteStalePendingUploads removes every unattached upload older than models.PendingUploadRetention,
// files first.
// An upload whose files or row cannot be removed is kept for the next run; the count and the joined
// failures report how the run went.
func DeleteStalePendingUploads(
	ctx context.Context,
	d DeleteStalePendingUploadsDependencies,
) (DeleteStalePendingUploadsResult, error) {
	staleUploads, err := d.UploadRepository.ListUnusedPendingUploads(ctx)
	if err != nil {
		return DeleteStalePendingUploadsResult{}, err
	}

	deletedCount := 0
	var failures []error

	for _, upload := range staleUploads {
		if err := deleteStaleUpload(ctx, d, upload); err != nil {
			failures = append(failures, err)
			continue
		}

		deletedCount++
	}

	return DeleteStalePendingUploadsResult{DeletedCount: deletedCount}, errors.Join(failures...)
}

func deleteStaleUpload(ctx context.Context, d DeleteStalePendingUploadsDependencies, upload models.Upload) error {
	for _, path := range upload.StoredPaths() {
		if err := deleteStoredFileIfPresent(ctx, d.FileStorage, path); err != nil {
			return err
		}
	}

	if err := d.UploadRepository.DeleteUploadByID(ctx, upload.ID); err != nil {
		return err
	}

	uploadevents.DispatchUploadDeleted(ctx, d.EventDispatcher, uploadevents.NewUploadDeletedParams{
		UploadID: upload.ID,
		UserID:   upload.UserID,
		Purpose:  string(upload.Purpose),
	})

	return nil
}

func deleteStoredFileIfPresent(ctx context.Context, fileStorage storage.FileStorage, path string) error {
	err := fileStorage.DeleteFile(ctx, path)
	if err == nil {
		return nil
	}
	if _, missing := errors.AsType[e.ErrStoredFileNotFound](err); missing {
		return nil
	}

	return e.ErrDeleteStoredFileFailed{Path: path, Cause: err}
}
