package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/auth/mappers"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
)

var _ refreshtoken.WriteRepository = RefreshTokenWriteRepository{}

// RefreshTokenWriteRepository stores refresh tokens through the generated sqlc queries.
type RefreshTokenWriteRepository struct {
	queries *sqlc.Queries
	days    int32
}

// NewRefreshTokenWriteRepository builds a repository that writes through the given sqlc queries.
func NewRefreshTokenWriteRepository(queries *sqlc.Queries) RefreshTokenWriteRepository {
	return RefreshTokenWriteRepository{queries: queries}
}

// Save inserts the refresh token and returns the row the database produced.
func (r RefreshTokenWriteRepository) Save(
	ctx context.Context,
	newRefreshToken refreshtoken.NewRefreshToken,
) (refreshtoken.RefreshToken, error) {
	row, err := r.queries.CreateRefreshToken(ctx, sqlc.CreateRefreshTokenParams{
		ID:        newRefreshToken.ID,
		UserID:    newRefreshToken.UserID,
		TokenHash: newRefreshToken.TokenHash,
		CreatedAt: time.Now(),
		ExpiresAt: newRefreshToken.ExpiresAt,
	})
	if err != nil {
		return refreshtoken.RefreshToken{}, refreshtoken.ErrSaveRefreshTokenFailed{Cause: err}
	}

	return mappers.RefreshTokenMapper(row), nil
}

// Revoke stamps the matching unrevoked token revoked and returns the row the database produced.
func (r RefreshTokenWriteRepository) Revoke(
	ctx context.Context,
	tokenHash string,
) (refreshtoken.RefreshToken, error) {
	row, err := r.queries.RevokeRefreshToken(ctx, tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return refreshtoken.RefreshToken{}, refreshtoken.ErrRefreshTokenNotFound{}
	}
	if err != nil {
		return refreshtoken.RefreshToken{}, refreshtoken.ErrRevokeRefreshTokenFailed{Cause: err}
	}

	return mappers.RefreshTokenMapper(row), nil
}

// DeleteAllForUser hard-deletes every refresh token the user holds, ending all their sessions.
func (r RefreshTokenWriteRepository) DeleteAllForUser(
	ctx context.Context,
	userID uuid.UUID,
) error {
	if err := r.queries.DeleteAllRefreshTokens(ctx, userID); err != nil {
		return refreshtoken.ErrDeleteRefreshTokensFailed{Cause: err}
	}

	return nil
}

// DeleteUnused purges expired tokens and ones revoked over days ago, returning the row count.
func (r RefreshTokenWriteRepository) DeleteUnused(ctx context.Context, days int32) (int64, error) {
	deletedCount, err := r.queries.DeleteUnusedRefreshTokens(ctx, days)
	if err != nil {
		return 0, refreshtoken.ErrDeleteUnusedRefreshTokensFailed{Cause: err}
	}

	return deletedCount, nil
}
