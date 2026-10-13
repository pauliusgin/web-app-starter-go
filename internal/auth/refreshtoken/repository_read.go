package refreshtoken

import "context"

// ReadRepository is the refresh token lookup port the domain depends on.
type ReadRepository interface {
	FindByTokenHash(ctx context.Context, tokenHash string) (RefreshToken, error)
}
