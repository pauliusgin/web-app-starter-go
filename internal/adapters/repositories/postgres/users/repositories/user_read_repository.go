package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/users/mappers"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/models"
	read "github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/repositories"
)

var _ read.UserRepository = UserReadRepository{}

// UserReadRepository looks users up through the generated sqlc queries.
type UserReadRepository struct {
	queries *sqlc.Queries
}

// NewUserReadRepository builds a repository that reads through the given sqlc queries.
func NewUserReadRepository(queries *sqlc.Queries) UserReadRepository {
	return UserReadRepository{queries: queries}
}

// GetUserReadModelByID returns the user with the given ID, or e.ErrUserNotFound.
func (r UserReadRepository) GetUserReadModelByID(ctx context.Context, id uuid.UUID) (models.UserReadModel, error) {
	row, err := r.queries.GetUserReadModelByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return models.UserReadModel{}, e.ErrUserNotFound{ID: id}
	}
	if err != nil {
		return models.UserReadModel{}, e.ErrFindUserFailed{ID: id, Cause: err}
	}

	return mappers.UserReadModelFromIDMapper(row), nil
}
