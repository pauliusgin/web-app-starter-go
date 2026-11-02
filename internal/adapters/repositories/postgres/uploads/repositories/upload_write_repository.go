// Package repositories adapts the generated sqlc queries to the uploads ports.
package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/uploads/mappers"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/models"
	write "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/repositories"
)

var _ write.UploadWriteRepository = UploadWriteRepository{}

// UploadWriteRepository records and purges uploads through the generated sqlc queries.
type UploadWriteRepository struct {
	queries *sqlc.Queries
}

// NewUploadWriteRepository builds a repository that works through the given sqlc queries.
func NewUploadWriteRepository(queries *sqlc.Queries) UploadWriteRepository {
	return UploadWriteRepository{queries: queries}
}

// SaveUpload inserts the upload as pending and returns the row the database produced.
func (r UploadWriteRepository) SaveUpload(ctx context.Context, newUpload models.NewUpload) (models.Upload, error) {
	row, err := r.queries.SaveUpload(ctx, mappers.SaveUploadParamsMapper(newUpload))
	if err != nil {
		return models.Upload{}, e.ErrSaveUploadFailed{ID: newUpload.ID, Cause: err}
	}

	return mappers.UploadMapper(row), nil
}

// ListUnusedPendingUploads returns unattached uploads older than models.PendingUploadRetention, oldest first.
func (r UploadWriteRepository) ListUnusedPendingUploads(
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

// ListOrphanedAttachedUploads returns attached uploads absent from the upload_references view, attached
// earlier than models.PendingUploadRetention ago, oldest attachment first.
func (r UploadWriteRepository) ListOrphanedAttachedUploads(ctx context.Context) ([]models.Upload, error) {
	cutoff := time.Now().Add(-models.PendingUploadRetention)

	rows, err := r.queries.ListOrphanedAttachedUploads(ctx, cutoff)
	if err != nil {
		return nil, e.ErrListOrphanedUploadsFailed{Cause: err}
	}

	uploads := make([]models.Upload, 0, len(rows))
	for _, row := range rows {
		uploads = append(uploads, mappers.UploadMapper(row))
	}

	return uploads, nil
}

// FindUnreferencedUploadByID returns the upload when the upload_references view does not list it, or
// e.ErrUploadNotFound when it is missing or still referenced.
func (r UploadWriteRepository) FindUnreferencedUploadByID(ctx context.Context, id uuid.UUID) (models.Upload, error) {
	row, err := r.queries.FindUnreferencedUploadByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Upload{}, e.ErrUploadNotFound{ID: id}
	}
	if err != nil {
		return models.Upload{}, e.ErrFindUploadFailed{ID: id, Cause: err}
	}

	return mappers.UploadMapper(row), nil
}

// DeleteUploadByID removes the upload row; deleting an id that is already gone is not an error.
func (r UploadWriteRepository) DeleteUploadByID(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteUploadByID(ctx, id); err != nil {
		return e.ErrDeleteUploadFailed{ID: id, Cause: err}
	}

	return nil
}

// AttachUpload stamps attached_at with NOW() on the upload matching id, owner and purpose, or returns
// e.ErrUploadNotFound when none matches.
func (r UploadWriteRepository) AttachUpload(
	ctx context.Context,
	params models.AttachUploadParams,
) (models.Upload, error) {
	row, err := r.queries.AttachUpload(ctx, mappers.AttachUploadParamsMapper(params))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Upload{}, e.ErrUploadNotFound{ID: params.ID}
	}
	if err != nil {
		return models.Upload{}, e.ErrAttachUploadFailed{ID: params.ID, Cause: err}
	}

	return mappers.UploadMapper(row), nil
}
