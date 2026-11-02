package refreshtoken

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
)

// RotateDependencies carries the token lookup and the store the old token is revoked and its replacement saved in.
type RotateDependencies struct {
	ReadRepository  ReadRepository
	WriteRepository WriteRepository
}

// RotateParams is the token the client presented.
type RotateParams struct {
	PresentedToken string
}

// RotateResult is the replacement token; minting the access token stays the API's job.
type RotateResult struct {
	UserID    uuid.UUID
	Plaintext string
	ExpiresAt time.Time
}

// Rotate revokes the presented token and issues its replacement to the same user.
func Rotate(
	ctx context.Context,
	d RotateDependencies,
	params RotateParams,
) (RotateResult, error) {
	storedToken, err := d.ReadRepository.FindByTokenHash(ctx, token.Hash(params.PresentedToken))
	if err != nil {
		return RotateResult{}, err
	}

	if _, err := d.WriteRepository.Revoke(ctx, storedToken.TokenHash); err != nil {
		return RotateResult{}, err
	}

	issued, err := Issue(ctx,
		IssueDependencies{WriteRepository: d.WriteRepository},
		IssueParams{UserID: storedToken.UserID})
	if err != nil {
		return RotateResult{}, err
	}

	return RotateResult{
		UserID:    storedToken.UserID,
		Plaintext: issued.Plaintext,
		ExpiresAt: issued.ExpiresAt,
	}, nil
}
