package usecases

import (
	"context"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	writerepo "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
)

// RequestEmailChangeParams is the authenticated user and the address they want.
type RequestEmailChangeParams struct {
	UserID   uuid.UUID
	NewEmail string
}

// RequestEmailChangeDependencies carries the collaborators requesting an email change needs.
type RequestEmailChangeDependencies struct {
	UserWriteRepository   writerepo.UserRepository
	GenerateAuthChallenge GenerateAuthChallenge
}

// RequestEmailChange mails a verification link to the new address once it is free and different.
func RequestEmailChange(
	ctx context.Context,
	d RequestEmailChangeDependencies,
	params RequestEmailChangeParams,
) error {
	storedUser, err := d.UserWriteRepository.FindUserByID(ctx, params.UserID)
	if err != nil {
		return err
	}

	if storedUser.Email == params.NewEmail {
		return e.ErrEmailUnchanged{Email: params.NewEmail}
	}

	if err := ensureEmailAvailable(ctx, d.UserWriteRepository, params.NewEmail); err != nil {
		return err
	}

	_, err = d.GenerateAuthChallenge(ctx, challenge.GenerateParams{
		UserID:     storedUser.ID,
		Target:     params.NewEmail,
		TargetType: challenge.TargetTypeEmail,
		Purpose:    challenge.PurposeEmailVerification,
	})

	return err
}
