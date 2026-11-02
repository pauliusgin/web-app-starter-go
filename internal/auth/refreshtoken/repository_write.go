package refreshtoken

import (
	"context"

	"github.com/google/uuid"
)

// WriteRepository is the refresh token persistence port the domain depends on.
type WriteRepository interface {
	// SaveRefreshToken records the token hash and returns the token as stored.
	SaveRefreshToken(ctx context.Context, newRefreshToken NewRefreshToken) (RefreshToken, error)

	// Revoke stamps the matching unrevoked token revoked, or returns ErrRefreshTokenNotFound.
	Revoke(ctx context.Context, tokenHash string) (RefreshToken, error)

	// DeleteAllForUser removes every refresh token the user holds, ending all their sessions.
	DeleteAllForUser(ctx context.Context, userID uuid.UUID) error

	// DeleteUnused purges expired tokens and ones revoked over revokedDays ago, returning how many it removed.
	DeleteUnused(ctx context.Context, revokedDays int32) (int64, error)
}
