package refreshtoken

import (
	"context"

	"github.com/google/uuid"
)

// WriteRepository is the refresh token persistence port the domain depends on.
type WriteRepository interface {
	Save(ctx context.Context, newRefreshToken NewRefreshToken) (RefreshToken, error)

	Revoke(ctx context.Context, tokenHash string) (RefreshToken, error)

	DeleteAllForUser(ctx context.Context, userID uuid.UUID) error

	DeleteUnused(ctx context.Context, days int32) (int64, error)
}
