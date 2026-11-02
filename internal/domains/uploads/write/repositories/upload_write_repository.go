// Package repositories declares the uploads table port the upload use cases depend on.
package repositories

import (
	"context"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/models"
)

// UploadWriteRepository is the uploads table port the upload use cases record and purge uploads through.
type UploadWriteRepository interface {
	// SaveUpload records the upload as pending and returns it as stored.
	SaveUpload(ctx context.Context, newUpload models.NewUpload) (models.Upload, error)

	// ListUnusedPendingUploads returns unattached uploads older than models.PendingUploadRetention, oldest first.
	ListUnusedPendingUploads(ctx context.Context) ([]models.Upload, error)

	// ListOrphanedAttachedUploads returns attached uploads nothing references any more, attached earlier than
	// models.PendingUploadRetention ago, oldest attachment first.
	ListOrphanedAttachedUploads(ctx context.Context) ([]models.Upload, error)
	// FindUnreferencedUploadByID returns the upload when nothing references it, or errors.ErrUploadNotFound
	// when it is missing or still referenced.
	FindUnreferencedUploadByID(ctx context.Context, id uuid.UUID) (models.Upload, error)
	// DeleteUploadByID removes the upload record; an id that is already gone is not an error.
	DeleteUploadByID(ctx context.Context, id uuid.UUID) error

	// AttachUpload marks the upload claimed, sparing it from cleanup, or returns errors.ErrUploadNotFound
	// when no upload matches the id, owner and purpose.
	AttachUpload(ctx context.Context, params models.AttachUploadParams) (models.Upload, error)
}
