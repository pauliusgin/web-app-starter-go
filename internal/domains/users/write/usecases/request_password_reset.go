package usecases

import (
	"context"
	"errors"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
)

// RequestPasswordResetParams is the email the caller says their forgotten password belongs to.
type RequestPasswordResetParams struct {
	Email string
}

// RequestPasswordResetDependencies carries the user store, the issuance limit check and the link generator.
type RequestPasswordResetDependencies struct {
	UserWriteRepository            repositories.UserRepository
	EnsureChallengeIssuanceAllowed EnsureChallengeIssuanceAllowed
	GenerateAuthChallenge          GenerateAuthChallenge
}

// RequestPasswordReset mails a password reset link to the email's password account. It answers nil without
// sending anything for an unknown email, a google-only account or a rate limited one, so the caller cannot
// learn which emails have accounts.
func RequestPasswordReset(
	ctx context.Context,
	d RequestPasswordResetDependencies,
	params RequestPasswordResetParams,
) error {
	credentials, resettable, err := findResettableCredentials(ctx, d.UserWriteRepository, params.Email)
	if err != nil || !resettable {
		return err
	}

	err = d.EnsureChallengeIssuanceAllowed(ctx, challenge.EnsureIssuanceAllowedParams{
		UserID:  credentials.ID,
		Target:  credentials.Email,
		Purpose: challenge.PurposePasswordReset,
	})
	if errors.As(err, &challenge.ErrIssuanceRateLimited{}) {
		return nil
	}
	if err != nil {
		return err
	}

	_, err = d.GenerateAuthChallenge(ctx, challenge.GenerateParams{
		UserID:     credentials.ID,
		Target:     credentials.Email,
		TargetType: challenge.TargetTypeEmail,
		Purpose:    challenge.PurposePasswordReset,
	})

	return err
}

func findResettableCredentials(
	ctx context.Context,
	userWriteRepository repositories.UserRepository,
	email string,
) (models.UserCredentials, bool, error) {
	credentials, err := userWriteRepository.FindUserCredentialsByEmail(ctx, email)
	if errors.As(err, &e.ErrUserNotFound{}) {
		return models.UserCredentials{}, false, nil
	}
	if err != nil {
		return models.UserCredentials{}, false, err
	}

	return credentials, credentials.HasPassword(), nil
}
