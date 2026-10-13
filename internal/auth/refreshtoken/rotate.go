package refreshtoken

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
)

// RotateDependencies carries the collaborators rotating a refresh token needs.
type RotateDependencies struct {
	ReadRepository  ReadRepository
	WriteRepository WriteRepository
}

// RotateParams is the token the client presented.
type RotateParams struct {
	PresentedToken string
}

// Rotated is the replacement token; minting the access token stays the API's job.
type Rotated struct {
	UserID    uuid.UUID
	Plaintext string
	ExpiresAt time.Time
}

// Rotate revokes the presented token and issues its replacement to the same user.
func Rotate(
	ctx context.Context,
	d RotateDependencies,
	params RotateParams,
) (Rotated, error) {
	storedToken, err := d.ReadRepository.FindByTokenHash(ctx, token.Hash(params.PresentedToken))
	if err != nil {
		return Rotated{}, err
	}

	if _, err := d.WriteRepository.Revoke(ctx, storedToken.TokenHash); err != nil {
		return Rotated{}, err
	}

	issued, err := Issue(ctx,
		IssuingDependencies{WriteRepository: d.WriteRepository},
		IssuingParams{UserID: storedToken.UserID})
	if err != nil {
		return Rotated{}, err
	}

	return Rotated{
		UserID:    storedToken.UserID,
		Plaintext: issued.Plaintext,
		ExpiresAt: issued.ExpiresAt,
	}, nil
}
