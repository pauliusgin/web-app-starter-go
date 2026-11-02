package refreshtoken

import (
	"context"
	"time"

	"github.com/google/uuid"
)

var _ WriteRepository = (*InMemoryRefreshTokenRepository)(nil)
var _ ReadRepository = (*InMemoryRefreshTokenRepository)(nil)

// InMemoryRefreshTokenRepository is both refresh token ports over one map keyed by token hash, answering as the
// refresh_tokens queries do; Now stands in for the database clock and reads the wall clock when nil.
type InMemoryRefreshTokenRepository struct {
	Tokens map[string]RefreshToken
	Now    func() time.Time

	// FindFailure, SaveFailure, RevokeFailure, DeleteFailure and DeleteUnusedFailure, when set, are returned
	// instead of doing the work.
	FindFailure         error
	SaveFailure         error
	RevokeFailure       error
	DeleteFailure       error
	DeleteUnusedFailure error
}

// NewInMemoryRefreshTokenRepository builds a repository holding the given tokens and no failure staged.
func NewInMemoryRefreshTokenRepository(tokens ...RefreshToken) *InMemoryRefreshTokenRepository {
	stored := make(map[string]RefreshToken, len(tokens))
	for _, refreshToken := range tokens {
		stored[refreshToken.TokenHash] = refreshToken
	}

	return &InMemoryRefreshTokenRepository{Tokens: stored}
}

// FindByTokenHash returns the unrevoked, unexpired token with the hash, or ErrRefreshTokenNotFound.
func (r *InMemoryRefreshTokenRepository) FindByTokenHash(_ context.Context, tokenHash string) (RefreshToken, error) {
	if r.FindFailure != nil {
		return RefreshToken{}, r.FindFailure
	}

	stored, found := r.Tokens[tokenHash]
	if !found || stored.IsRevoked() || stored.IsExpired(r.now()) {
		return RefreshToken{}, ErrRefreshTokenNotFound{}
	}

	return stored, nil
}

// SaveRefreshToken records the token stamped with the repository clock, or returns SaveFailure when staged.
func (r *InMemoryRefreshTokenRepository) SaveRefreshToken(
	_ context.Context,
	newRefreshToken NewRefreshToken,
) (RefreshToken, error) {
	if r.SaveFailure != nil {
		return RefreshToken{}, r.SaveFailure
	}

	saved := RefreshToken{
		ID:        newRefreshToken.ID,
		UserID:    newRefreshToken.UserID,
		TokenHash: newRefreshToken.TokenHash,
		CreatedAt: r.now(),
		ExpiresAt: newRefreshToken.ExpiresAt,
	}
	r.Tokens[saved.TokenHash] = saved

	return saved, nil
}

// Revoke stamps the matching unrevoked token revoked, or returns ErrRefreshTokenNotFound.
func (r *InMemoryRefreshTokenRepository) Revoke(_ context.Context, tokenHash string) (RefreshToken, error) {
	if r.RevokeFailure != nil {
		return RefreshToken{}, r.RevokeFailure
	}

	stored, found := r.Tokens[tokenHash]
	if !found || stored.IsRevoked() {
		return RefreshToken{}, ErrRefreshTokenNotFound{}
	}

	stored.RevokedAt = r.now()
	r.Tokens[tokenHash] = stored

	return stored, nil
}

// DeleteAllForUser removes every token the user holds, ending all their sessions.
func (r *InMemoryRefreshTokenRepository) DeleteAllForUser(_ context.Context, userID uuid.UUID) error {
	if r.DeleteFailure != nil {
		return r.DeleteFailure
	}

	for tokenHash, stored := range r.Tokens {
		if stored.UserID == userID {
			delete(r.Tokens, tokenHash)
		}
	}

	return nil
}

// DeleteUnused purges expired tokens and ones revoked over revokedDays ago, returning how many it removed.
func (r *InMemoryRefreshTokenRepository) DeleteUnused(_ context.Context, revokedDays int32) (int64, error) {
	if r.DeleteUnusedFailure != nil {
		return 0, r.DeleteUnusedFailure
	}

	now := r.now()
	revokedBefore := now.AddDate(0, 0, -int(revokedDays))

	deletedCount := int64(0)
	for tokenHash, stored := range r.Tokens {
		expired := stored.ExpiresAt.Before(now)
		revokedLongAgo := stored.IsRevoked() && stored.RevokedAt.Before(revokedBefore)
		if expired || revokedLongAgo {
			delete(r.Tokens, tokenHash)
			deletedCount++
		}
	}

	return deletedCount, nil
}

func (r *InMemoryRefreshTokenRepository) now() time.Time {
	if r.Now == nil {
		return time.Now()
	}

	return r.Now()
}
