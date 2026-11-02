package challenge

import (
	"context"

	"github.com/google/uuid"
)

// WriteRepository is the challenge persistence port the domain depends on.
type WriteRepository interface {
	// SaveChallenge records the challenge and returns it as stored.
	SaveChallenge(ctx context.Context, newChallenge NewChallenge) (Challenge, error)

	// MarkUsed spends the challenge so its link cannot be replayed; an already spent one is not an error.
	MarkUsed(ctx context.Context, challengeID uuid.UUID) error
}
