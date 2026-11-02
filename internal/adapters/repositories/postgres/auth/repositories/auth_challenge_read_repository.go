// Package repositories adapts the sqlc queries to the auth challenge and refresh token ports.
package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/auth/mappers"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlnull"
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
	row, err := r.queries.FindAuthChallengeByTokenHash(ctx, sqlnull.StringUnlessEmpty(tokenHash))
	if errors.Is(err, sql.ErrNoRows) {
		return challenge.Challenge{}, challenge.ErrChallengeNotFound{}
	}
	if err != nil {
		return challenge.Challenge{}, challenge.ErrChallengeLookupFailed{Cause: err}
	}

	return mappers.AuthChallengeMapper(row), nil
}

// ListCreatedAtByTarget returns, oldest first, when the user sent challenges for the purpose to the target.
func (r AuthChallengeReadRepository) ListCreatedAtByTarget(
	ctx context.Context,
	params challenge.ListCreatedAtByTargetParams,
) ([]time.Time, error) {
	createdAt, err := r.queries.ListAuthChallengeCreatedAtByTarget(ctx, sqlc.ListAuthChallengeCreatedAtByTargetParams{
		UserID:       params.UserID,
		Target:       params.Target,
		Purpose:      string(params.Purpose),
		CreatedAfter: params.CreatedAfter,
	})
	if err != nil {
		return nil, challenge.ErrIssuanceHistoryLookupFailed{Cause: err}
	}

	return createdAt, nil
}

// ListCreatedAtByUser returns, oldest first, when challenges for the purpose were minted for the user.
func (r AuthChallengeReadRepository) ListCreatedAtByUser(
	ctx context.Context,
	params challenge.ListCreatedAtByUserParams,
) ([]time.Time, error) {
	createdAt, err := r.queries.ListAuthChallengeCreatedAtByUser(ctx, sqlc.ListAuthChallengeCreatedAtByUserParams{
		UserID:       params.UserID,
		Purpose:      string(params.Purpose),
		CreatedAfter: params.CreatedAfter,
	})
	if err != nil {
		return nil, challenge.ErrIssuanceHistoryLookupFailed{Cause: err}
	}

	return createdAt, nil
}
