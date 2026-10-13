package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/auth/mappers"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
)

var _ refreshtoken.ReadRepository = RefreshTokenReadRepository{}

// RefreshTokenReadRepository looks refresh tokens up through the generated sqlc queries.
type RefreshTokenReadRepository struct {
	queries *sqlc.Queries
}

// NewRefreshTokenReadRepository builds a repository that reads through the given sqlc queries.
func NewRefreshTokenReadRepository(queries *sqlc.Queries) RefreshTokenReadRepository {
	return RefreshTokenReadRepository{queries: queries}
}

// FindByTokenHash returns only non-revoked, non-expired refresh tokens.
func (r RefreshTokenReadRepository) FindByTokenHash(
	ctx context.Context,
	tokenHash string,
) (refreshtoken.RefreshToken, error) {
	row, err := r.queries.FindRefreshTokenByTokenHash(ctx, tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return refreshtoken.RefreshToken{}, refreshtoken.ErrRefreshTokenNotFound{}
	}
	if err != nil {
		return refreshtoken.RefreshToken{}, refreshtoken.ErrRefreshTokenLookupFailed{Cause: err}
	}

	return mappers.RefreshTokenMapper(row), nil
}
