package refreshtoken

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
)

var fixedCreatedAt = time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)

var fixedRotatedAt = time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)

var fixedRevokedAt = time.Date(2026, time.September, 6, 12, 0, 1, 0, time.UTC)

const cleanUpThresholdInDays = 7

type inMemoryTokenStore struct {
	rows                map[string]RefreshToken
	readAt              time.Time
	lookupFailure       error
	saveFailure         error
	revokeFailure       error
	deleteFailure       error
	deleteUnusedFailure error
}

func newInMemoryTokenStore() *inMemoryTokenStore {
	return &inMemoryTokenStore{rows: map[string]RefreshToken{}, readAt: fixedRotatedAt}
}

func (s *inMemoryTokenStore) FindByTokenHash(
	_ context.Context,
	tokenHash string,
) (RefreshToken, error) {
	if s.lookupFailure != nil {
		return RefreshToken{}, s.lookupFailure
	}

	storedToken, found := s.rows[tokenHash]
	if !found || storedToken.IsRevoked() || storedToken.IsExpired(s.readAt) {
		return RefreshToken{}, ErrRefreshTokenNotFound{}
	}

	return storedToken, nil
}

func (s *inMemoryTokenStore) Save(
	_ context.Context,
	newRefreshToken NewRefreshToken,
) (RefreshToken, error) {
	if s.saveFailure != nil {
		return RefreshToken{}, s.saveFailure
	}

	storedToken := RefreshToken{
		ID:        newRefreshToken.ID,
		UserID:    newRefreshToken.UserID,
		TokenHash: newRefreshToken.TokenHash,
		CreatedAt: fixedCreatedAt,
		ExpiresAt: newRefreshToken.ExpiresAt,
	}
	s.rows[storedToken.TokenHash] = storedToken

	return storedToken, nil
}

func (s *inMemoryTokenStore) Revoke(
	_ context.Context,
	tokenHash string,
) (RefreshToken, error) {
	if s.revokeFailure != nil {
		return RefreshToken{}, s.revokeFailure
	}

	storedToken, found := s.rows[tokenHash]
	if !found || storedToken.IsRevoked() {
		return RefreshToken{}, ErrRefreshTokenNotFound{}
	}

	storedToken.RevokedAt = fixedRevokedAt
	s.rows[tokenHash] = storedToken

	return storedToken, nil
}

func (s *inMemoryTokenStore) DeleteAllForUser(_ context.Context, userID uuid.UUID) error {
	if s.deleteFailure != nil {
		return s.deleteFailure
	}

	for tokenHash, storedToken := range s.rows {
		if storedToken.UserID == userID {
			delete(s.rows, tokenHash)
		}
	}

	return nil
}

func (s *inMemoryTokenStore) DeleteUnused(_ context.Context, days int32) (int64, error) {
	if s.deleteUnusedFailure != nil {
		return 0, s.deleteUnusedFailure
	}

	graceExpiresAt := s.readAt.AddDate(0, 0, -int(days))

	deletedCount := int64(0)
	for tokenHash, storedToken := range s.rows {
		expired := storedToken.ExpiresAt.Before(s.readAt)
		revokedBeyondGrace := storedToken.IsRevoked() &&
			storedToken.RevokedAt.Before(graceExpiresAt)

		if expired || revokedBeyondGrace {
			delete(s.rows, tokenHash)
			deletedCount++
		}
	}

	return deletedCount, nil
}

type seedTokenParams struct {
	UserID    uuid.UUID
	ExpiresAt time.Time
	RevokedAt time.Time
}

func (s *inMemoryTokenStore) seedToken(t *testing.T, params seedTokenParams) string {
	t.Helper()

	minted := token.Generate()

	expiresAt := params.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = fixedRotatedAt.Add(Lifetime)
	}

	s.rows[minted.Hash] = RefreshToken{
		ID:        uuid.New(),
		UserID:    params.UserID,
		TokenHash: minted.Hash,
		CreatedAt: fixedCreatedAt,
		ExpiresAt: expiresAt,
		RevokedAt: params.RevokedAt,
	}

	return minted.Plaintext
}

func (s *inMemoryTokenStore) tokensForUser(userID uuid.UUID) []RefreshToken {
	var found []RefreshToken

	for _, storedToken := range s.rows {
		if storedToken.UserID == userID {
			found = append(found, storedToken)
		}
	}

	return found
}

func (s *inMemoryTokenStore) tokenByPlaintext(
	t *testing.T,
	plaintext string,
) RefreshToken {
	t.Helper()

	storedToken, found := s.rows[token.Hash(plaintext)]
	if !found {
		t.Fatalf("no stored row for the presented token")
	}

	return storedToken
}
