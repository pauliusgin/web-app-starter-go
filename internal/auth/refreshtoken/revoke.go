package refreshtoken

import (
	"context"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
)

// RevokeDependencies carries the collaborators revoking a refresh token needs.
type RevokeDependencies struct {
	ReadRepository  ReadRepository
	WriteRepository WriteRepository
}

// RevokeParams is the refresh token the logging-out client presented.
type RevokeParams struct {
	PresentedToken string
}

// Revoke ends the presented session.
func Revoke(
	ctx context.Context,
	d RevokeDependencies,
	params RevokeParams,
) error {
	storedToken, err := d.ReadRepository.FindByTokenHash(ctx, token.Hash(params.PresentedToken))
	if err != nil {
		return err
	}

	if _, err := d.WriteRepository.Revoke(ctx, storedToken.TokenHash); err != nil {
		return err
	}

	return nil
}
