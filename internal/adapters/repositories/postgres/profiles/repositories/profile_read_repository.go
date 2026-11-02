package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/profiles/mappers"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/read/models"
	read "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/read/repositories"
)

var _ read.ProfileReadRepository = ProfileReadRepository{}

// ProfileReadRepository looks client-facing profiles up through the generated sqlc queries.
type ProfileReadRepository struct {
	queries *sqlc.Queries
}

// NewProfileReadRepository builds a repository that reads through the given sqlc queries.
func NewProfileReadRepository(queries *sqlc.Queries) ProfileReadRepository {
	return ProfileReadRepository{queries: queries}
}

// GetProfileReadModelByID returns the client-facing view of the user's live profile, or e.ErrProfileNotFound.
func (r ProfileReadRepository) GetProfileReadModelByID(
	ctx context.Context,
	userID uuid.UUID,
) (models.ProfileReadModel, error) {
	row, err := r.queries.GetProfileReadModelByID(ctx, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return models.ProfileReadModel{}, e.ErrProfileNotFound{UserID: userID}
	}
	if err != nil {
		return models.ProfileReadModel{}, e.ErrFindProfileFailed{UserID: userID, Cause: err}
	}

	return mappers.ProfileReadModelFromIDMapper(row), nil
}
