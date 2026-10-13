package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/auth/mappers"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
)

var _ challenge.WriteRepository = AuthChallengeWriteRepository{}

// AuthChallengeWriteRepository stores auth challenges through the generated sqlc queries.
type AuthChallengeWriteRepository struct {
	queries *sqlc.Queries
}

// NewAuthChallengeWriteRepository builds a repository that writes through the given sqlc queries.
func NewAuthChallengeWriteRepository(queries *sqlc.Queries) AuthChallengeWriteRepository {
	return AuthChallengeWriteRepository{queries: queries}
}

// Save inserts the auth challenge and returns the row the database produced.
func (r AuthChallengeWriteRepository) Save(
	ctx context.Context,
	newChallenge challenge.NewChallenge,
) (challenge.Challenge, error) {
	row, err := r.queries.CreateAuthChallenge(ctx, sqlc.CreateAuthChallengeParams{
		ID:          newChallenge.ID,
		UserID:      newChallenge.UserID,
		TargetType:  string(newChallenge.TargetType),
		Target:      newChallenge.Target,
		Purpose:     string(newChallenge.Purpose),
		TokenHash:   sql.NullString{String: newChallenge.TokenHash, Valid: true},
		Code:        sql.NullString{},
		Provider:    sql.NullString{},
		ReferenceID: sql.NullString{},
		CreatedAt:   time.Now(),
		ExpiresAt:   newChallenge.ExpiresAt,
	})
	if err != nil {
		return challenge.Challenge{}, challenge.ErrSaveChallengeFailed{Cause: err}
	}

	return mappers.AuthChallengeMapper(row), nil
}

// MarkUsed spends the challenge so its link cannot be replayed.
func (r AuthChallengeWriteRepository) MarkUsed(ctx context.Context, challengeID uuid.UUID) error {
	_, err := r.queries.MarkAuthChallengeAsUsed(ctx, challengeID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return challenge.ErrConsumeChallengeFailed{Cause: err}
	}

	return nil
}
