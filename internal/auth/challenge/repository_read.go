package challenge

import "context"

// ReadRepository is the challenge lookup port the domain depends on.
type ReadRepository interface {
	FindByTokenHash(ctx context.Context, tokenHash string) (Challenge, error)
}
