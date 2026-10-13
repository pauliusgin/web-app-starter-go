// Package repositories adapts the generated sqlc queries to the uploads ports.
package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/uploads/mappers"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/repositories"
)

var _ repositories.UploadRepository = UploadRepository{}

// UploadRepository records and purges uploads through the generated sqlc queries.
type UploadRepository struct {
	queries *sqlc.Queries
}

// NewUploadRepository builds a repository that works through the given sqlc queries.
func NewUploadRepository(queries *sqlc.Queries) UploadRepository {
	return UploadRepository{queries: queries}
}

// SaveUpload inserts the upload as pending and returns the row the database produced.
func (r UploadRepository) SaveUpload(ctx context.Context, newUpload models.NewUpload) (models.Upload, error) {
	row, err := r.queries.SaveUpload(ctx, mappers.SaveUploadParamsMapper(newUpload))
	if err != nil {
		return models.Upload{}, e.ErrSaveUploadFailed{ID: newUpload.ID, Cause: err}
	}

	return mappers.UploadMapper(row), nil
}

// ListUnusedPendingUploads returns unattached uploads created before cutoff, oldest first.
func (r UploadRepository) ListUnusedPendingUploads(
	ctx context.Context,
) ([]models.Upload, error) {
	cutoff := time.Now().Add(-models.PendingUploadRetention)

	rows, err := r.queries.ListUnusedPendingUploads(ctx, cutoff)
	if err != nil {
		return nil, e.ErrListPendingUploadsFailed{Cause: err}
	}

	uploads := make([]models.Upload, 0, len(rows))
	for _, row := range rows {
		uploads = append(uploads, mappers.UploadMapper(row))
	}

	return uploads, nil
}

// DeleteUploadByID removes the upload row; deleting an id that is already gone is not an error.
func (r UploadRepository) DeleteUploadByID(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteUploadByID(ctx, id); err != nil {
		return e.ErrDeleteUploadFailed{ID: id, Cause: err}
	}

	return nil
}
