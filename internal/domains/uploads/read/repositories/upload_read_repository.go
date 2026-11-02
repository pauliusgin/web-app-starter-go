// Package repositories declares the upload lookup port the read side depends on.
package repositories

import (
	"context"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/read/models"
)

// UploadReadRepository is the upload lookup port the upload queries answer clients through.
type UploadReadRepository interface {
	// GetUploadReadModelByID returns the client-facing view of the upload owned by the user,
	// or errors.ErrUploadNotFound.
	GetUploadReadModelByID(ctx context.Context, params models.GetUploadReadModelByIDParams) (models.UploadReadModel, error)
}
