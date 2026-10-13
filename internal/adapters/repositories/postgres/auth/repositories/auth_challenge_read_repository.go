// Package repositories adapts the sqlc queries to the auth challenge and refresh token ports.
package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/auth/mappers"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
)

var _ challenge.ReadRepository = AuthChallengeReadRepository{}

// AuthChallengeReadRepository looks auth challenges up through the generated sqlc queries.
type AuthChallengeReadRepository struct {
	queries *sqlc.Queries
}

// NewAuthChallengeReadRepository builds a repository that reads through the given sqlc queries.
func NewAuthChallengeReadRepository(queries *sqlc.Queries) AuthChallengeReadRepository {
	return AuthChallengeReadRepository{queries: queries}
}

// FindByTokenHash returns only a challenge that is neither used nor expired.
func (r AuthChallengeReadRepository) FindByTokenHash(
	ctx context.Context,
	tokenHash string,
) (challenge.Challenge, error) {
	row, err := r.queries.FindAuthChallengeByTokenHash(ctx,
		sql.NullString{String: tokenHash, Valid: true})
	if errors.Is(err, sql.ErrNoRows) {
		return challenge.Challenge{}, challenge.ErrChallengeNotFound{}
	}
	if err != nil {
		return challenge.Challenge{}, challenge.ErrChallengeLookupFailed{Cause: err}
	}

	return mappers.AuthChallengeMapper(row), nil
}
