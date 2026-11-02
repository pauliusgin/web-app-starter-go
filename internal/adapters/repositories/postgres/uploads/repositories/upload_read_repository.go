package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/uploads/mappers"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/read/models"
	read "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/read/repositories"
)

var _ read.UploadReadRepository = UploadReadRepository{}

// UploadReadRepository looks client-facing uploads up through the generated sqlc queries.
type UploadReadRepository struct {
	queries *sqlc.Queries
}

// NewUploadReadRepository builds a repository that reads through the given sqlc queries.
func NewUploadReadRepository(queries *sqlc.Queries) UploadReadRepository {
	return UploadReadRepository{queries: queries}
}

// GetUploadReadModelByID returns the client-facing view of the upload owned by the user, or e.ErrUploadNotFound.
func (r UploadReadRepository) GetUploadReadModelByID(
	ctx context.Context,
	params models.GetUploadReadModelByIDParams,
) (models.UploadReadModel, error) {
	row, err := r.queries.GetUploadReadModelByID(ctx, sqlc.GetUploadReadModelByIDParams{
		ID:     params.ID,
		UserID: params.UserID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return models.UploadReadModel{}, e.ErrUploadNotFound{ID: params.ID}
	}
	if err != nil {
		return models.UploadReadModel{}, e.ErrFindUploadFailed{ID: params.ID, Cause: err}
	}

	return mappers.UploadReadModelFromIDMapper(row), nil
}
