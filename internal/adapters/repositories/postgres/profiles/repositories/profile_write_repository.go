// Package repositories adapts the generated sqlc queries to the profiles ports.
package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/profiles/mappers"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/models"
	write "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/repositories"
)

var _ write.ProfileWriteRepository = ProfileWriteRepository{}

// ProfileWriteRepository records and changes profiles through the generated sqlc queries.
type ProfileWriteRepository struct {
	queries *sqlc.Queries
}

// NewProfileWriteRepository builds a repository that works through the given sqlc queries.
func NewProfileWriteRepository(queries *sqlc.Queries) ProfileWriteRepository {
	return ProfileWriteRepository{queries: queries}
}

// SaveProfile inserts the profile, or returns e.ErrProfileAlreadyExists when the user has a row.
func (r ProfileWriteRepository) SaveProfile(ctx context.Context, newProfile models.NewProfile) (models.Profile, error) {
	row, err := r.queries.SaveProfile(ctx, mappers.SaveProfileParamsMapper(newProfile))
	if postgres.IsUniqueViolation(err) {
		return models.Profile{}, e.ErrProfileAlreadyExists{UserID: newProfile.ID}
	}
	if err != nil {
		return models.Profile{}, e.ErrSaveProfileFailed{UserID: newProfile.ID, Cause: err}
	}

	return mappers.ProfileMapper(row), nil
}

// UpdateProfile replaces the names and picture of a live profile, or returns e.ErrProfileNotFound.
func (r ProfileWriteRepository) UpdateProfile(
	ctx context.Context,
	change models.UpdateProfileParams,
) (models.Profile, error) {
	row, err := r.queries.UpdateProfile(ctx, mappers.UpdateProfileParamsMapper(change))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Profile{}, e.ErrProfileNotFound{UserID: change.ID}
	}
	if err != nil {
		return models.Profile{}, e.ErrUpdateProfileFailed{UserID: change.ID, Cause: err}
	}

	return mappers.ProfileMapper(row), nil
}
