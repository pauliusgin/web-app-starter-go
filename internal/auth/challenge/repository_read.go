package challenge

import "context"

// ReadRepository is the challenge lookup port the domain depends on.
type ReadRepository interface {
	// FindByTokenHash returns the unused, unexpired challenge with the hash, or ErrChallengeNotFound.
	FindByTokenHash(ctx context.Context, tokenHash string) (Challenge, error)
}
