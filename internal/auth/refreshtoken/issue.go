package refreshtoken

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
)

// IssueDependencies carries the refresh token store the new token is saved in.
type IssueDependencies struct {
	WriteRepository WriteRepository
}

// IssueParams is the user the token belongs to.
type IssueParams struct {
	UserID uuid.UUID
}

// IssueResult is what the caller receives; Plaintext is the only copy that will ever exist.
type IssueResult struct {
	Plaintext string
	ExpiresAt time.Time
}

// Issue mints a token, stores its hash, and hands the plaintext back to the caller.
func Issue(
	ctx context.Context,
	d IssueDependencies,
	params IssueParams,
) (IssueResult, error) {
	minted := token.Generate()

	expiresAt := time.Now().Add(Lifetime)

	_, err := d.WriteRepository.SaveRefreshToken(ctx, NewRefreshToken{
		ID:        uuid.New(),
		UserID:    params.UserID,
		TokenHash: minted.Hash,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return IssueResult{}, err
	}

	return IssueResult{Plaintext: minted.Plaintext, ExpiresAt: expiresAt}, nil
}
