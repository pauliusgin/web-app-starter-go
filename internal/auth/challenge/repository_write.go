package challenge

import (
	"context"

	"github.com/google/uuid"
)

// WriteRepository is the challenge persistence port the domain depends on.
type WriteRepository interface {
	Save(ctx context.Context, newChallenge NewChallenge) (Challenge, error)

	MarkUsed(ctx context.Context, challengeID uuid.UUID) error
}
