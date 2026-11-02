package usecases

import (
	"context"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
)

// ResendEmailVerificationParams is the authenticated user whose current email needs a fresh link.
type ResendEmailVerificationParams struct {
	UserID uuid.UUID
}

// EnsureChallengeIssuanceAllowed is the rate limit check run before asking for a challenge.
type EnsureChallengeIssuanceAllowed func(ctx context.Context, params challenge.EnsureIssuanceAllowedParams) error

// ResendEmailVerificationDependencies carries the user store, the issuance limit check and the link generator.
type ResendEmailVerificationDependencies struct {
	UserWriteRepository            repositories.UserRepository
	EnsureChallengeIssuanceAllowed EnsureChallengeIssuanceAllowed
	GenerateAuthChallenge          GenerateAuthChallenge
}

// ResendEmailVerification mails a new verification link to the user's still unverified current email.
func ResendEmailVerification(
	ctx context.Context,
	d ResendEmailVerificationDependencies,
	params ResendEmailVerificationParams,
) error {
	storedUser, err := d.UserWriteRepository.FindUserByID(ctx, params.UserID)
	if err != nil {
		return err
	}

	if storedUser.IsEmailVerified() {
		return e.ErrEmailAlreadyVerified{Email: storedUser.Email}
	}

	err = d.EnsureChallengeIssuanceAllowed(ctx, challenge.EnsureIssuanceAllowedParams{
		UserID:  storedUser.ID,
		Target:  storedUser.Email,
		Purpose: challenge.PurposeEmailVerification,
	})
	if err != nil {
		return err
	}

	_, err = d.GenerateAuthChallenge(ctx, challenge.GenerateParams{
		UserID:     storedUser.ID,
		Target:     storedUser.Email,
		TargetType: challenge.TargetTypeEmail,
		Purpose:    challenge.PurposeEmailVerification,
	})

	return err
}
