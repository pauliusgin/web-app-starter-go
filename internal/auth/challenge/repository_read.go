package challenge

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ReadRepository is the challenge lookup port the domain depends on.
type ReadRepository interface {
	// FindByTokenHash returns the unused, unexpired challenge with the hash, or ErrChallengeNotFound.
	FindByTokenHash(ctx context.Context, tokenHash string) (Challenge, error)

	// ListCreatedAtByTarget returns, oldest first, when challenges for the purpose were sent to the target;
	// the issuance limits read the last element as the latest.
	ListCreatedAtByTarget(ctx context.Context, params ListCreatedAtByTargetParams) ([]time.Time, error)

	// ListCreatedAtByUser returns, oldest first, when challenges for the purpose were minted for the user.
	ListCreatedAtByUser(ctx context.Context, params ListCreatedAtByUserParams) ([]time.Time, error)
}

// ListCreatedAtByTargetParams narrows the history to one user, target and purpose after an instant.
type ListCreatedAtByTargetParams struct {
	UserID       uuid.UUID
	Target       string
	Purpose      Purpose
	CreatedAfter time.Time
}

// ListCreatedAtByUserParams narrows the history to one user and purpose after an instant.
type ListCreatedAtByUserParams struct {
	UserID       uuid.UUID
	Purpose      Purpose
	CreatedAfter time.Time
}
