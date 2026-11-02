package refreshtoken

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
)

var fixedCreatedAt = time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)

var fixedRotatedAt = time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)

var fixedRevokedAt = time.Date(2026, time.September, 6, 12, 0, 1, 0, time.UTC)

const cleanUpThresholdInDays = 7

// newInMemoryTokenStore builds a store whose clock reads fixedRotatedAt, the instant the tests rotate at.
func newInMemoryTokenStore() *InMemoryRefreshTokenRepository {
	store := NewInMemoryRefreshTokenRepository()
	store.Now = func() time.Time { return fixedRotatedAt }

	return store
}

type seedTokenParams struct {
	UserID    uuid.UUID
	ExpiresAt time.Time
	RevokedAt time.Time
}

func seedToken(t *testing.T, store *InMemoryRefreshTokenRepository, params seedTokenParams) string {
	t.Helper()

	minted := token.Generate()

	expiresAt := params.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = fixedRotatedAt.Add(Lifetime)
	}

	store.Tokens[minted.Hash] = RefreshToken{
		ID:        uuid.New(),
		UserID:    params.UserID,
		TokenHash: minted.Hash,
		CreatedAt: fixedCreatedAt,
		ExpiresAt: expiresAt,
		RevokedAt: params.RevokedAt,
	}

	return minted.Plaintext
}

func tokensForUser(store *InMemoryRefreshTokenRepository, userID uuid.UUID) []RefreshToken {
	var found []RefreshToken

	for _, storedToken := range store.Tokens {
		if storedToken.UserID == userID {
			found = append(found, storedToken)
		}
	}

	return found
}

func tokenByPlaintext(t *testing.T, store *InMemoryRefreshTokenRepository, plaintext string) RefreshToken {
	t.Helper()

	storedToken, found := store.Tokens[token.Hash(plaintext)]
	if !found {
		t.Fatalf("no stored row for the presented token")
	}

	return storedToken
}
