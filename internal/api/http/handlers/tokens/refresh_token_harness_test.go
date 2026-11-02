package tokens

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
)

const presentedRefreshTokenForTest = "Xh2q8kU1n0dJ6oQpZ4tR7yC3vB5mN9wL1aS8dF0gH2k"

const unknownRefreshTokenForTest = "Qm3w9eR5tY1uI7oP2aS6dF0gH4jK8lZ3xC5vB7nM9q1"

// liveTokenExpiresAt sits far past the wall clock the repository checks the seeded token against.
var liveTokenExpiresAt = time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)

var tokenHolderForTest = models.User{
	ID:    uuid.MustParse("5b0c7d4e-2f8a-4c61-9e3b-7a1d6f0e8c29"),
	Email: "holder@example.com",
}

// newRefreshTokenStore holds one live token, opened by presentedRefreshTokenForTest, for tokenHolderForTest.
func newRefreshTokenStore() *refreshtoken.InMemoryRefreshTokenRepository {
	return refreshtoken.NewInMemoryRefreshTokenRepository(refreshtoken.RefreshToken{
		ID:        uuid.New(),
		UserID:    tokenHolderForTest.ID,
		TokenHash: token.Hash(presentedRefreshTokenForTest),
		ExpiresAt: liveTokenExpiresAt,
	})
}

func rotateOver(store *refreshtoken.InMemoryRefreshTokenRepository) RotateRefreshTokenUseCase {
	dependencies := refreshtoken.RotateDependencies{ReadRepository: store, WriteRepository: store}

	return func(ctx context.Context, params refreshtoken.RotateParams) (refreshtoken.RotateResult, error) {
		return refreshtoken.Rotate(ctx, dependencies, params)
	}
}

func revokeOver(store *refreshtoken.InMemoryRefreshTokenRepository) RevokeRefreshTokenUseCase {
	dependencies := refreshtoken.RevokeDependencies{ReadRepository: store, WriteRepository: store}

	return func(ctx context.Context, params refreshtoken.RevokeParams) error {
		return refreshtoken.Revoke(ctx, dependencies, params)
	}
}

func assertPresentedTokenRevoked(t *testing.T, store *refreshtoken.InMemoryRefreshTokenRepository, wantRevoked bool) {
	t.Helper()

	presented := store.Tokens[token.Hash(presentedRefreshTokenForTest)]
	if presented.IsRevoked() != wantRevoked {
		t.Errorf("presented token revoked = %t, want %t", presented.IsRevoked(), wantRevoked)
	}
}

func silenceRefreshTokenLogger(t *testing.T) {
	t.Helper()

	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
}
