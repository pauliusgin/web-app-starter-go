// Package repositories declares the uploads table port the upload use cases depend on.
package repositories

import (
	"context"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/models"
)

// UploadRepository is the uploads table port the upload use cases record and purge uploads through.
type UploadRepository interface {
	SaveUpload(ctx context.Context, newUpload models.NewUpload) (models.Upload, error)

	ListUnusedPendingUploads(ctx context.Context) ([]models.Upload, error)

	DeleteUploadByID(ctx context.Context, id uuid.UUID) error
}
