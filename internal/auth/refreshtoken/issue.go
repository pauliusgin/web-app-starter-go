package refreshtoken

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
)

// IssuingDependencies carries the collaborators issuing a refresh token needs.
type IssuingDependencies struct {
	WriteRepository WriteRepository
}

// IssuingParams is the user the token belongs to.
type IssuingParams struct {
	UserID uuid.UUID
}

// Issued is what the caller receives; Plaintext is the only copy that will ever exist.
type Issued struct {
	Plaintext string
	ExpiresAt time.Time
}

// Issue mints a token, stores its hash, and hands the plaintext back to the caller.
func Issue(
	ctx context.Context,
	d IssuingDependencies,
	params IssuingParams,
) (Issued, error) {
	minted := token.Generate()

	expiresAt := time.Now().Add(Lifetime)

	_, err := d.WriteRepository.Save(ctx, NewRefreshToken{
		ID:        uuid.New(),
		UserID:    params.UserID,
		TokenHash: minted.Hash,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return Issued{}, err
	}

	return Issued{Plaintext: minted.Plaintext, ExpiresAt: expiresAt}, nil
}
