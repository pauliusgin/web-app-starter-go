package usecases

import (
	"context"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/password"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	writerepo "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	userevents "github.com/pauliusgin/web_app_starter-be/internal/events/user"
)

// ChangePasswordParams is the authenticated user, the password they prove and the one they want.
type ChangePasswordParams struct {
	UserID          uuid.UUID
	CurrentPassword string
	NewPassword     string
}

// ChangePasswordDependencies carries the collaborators changing a password needs.
type ChangePasswordDependencies struct {
	UserWriteRepository writerepo.UserRepository
	EventDispatcher     events.Dispatcher
}

// ChangePassword stores a new password hash once the current password matches, then announces it.
func ChangePassword(ctx context.Context, d ChangePasswordDependencies, params ChangePasswordParams) error {
	credentials, err := d.UserWriteRepository.FindUserCredentialsByID(ctx, params.UserID)
	if err != nil {
		return err
	}

	if err := ensurePasswordReplaceable(credentials.PasswordHash, params); err != nil {
		return err
	}

	newPasswordHash, err := password.Hash(params.NewPassword)
	if err != nil {
		return e.ErrPasswordHashingFailed{Cause: err}
	}

	err = d.UserWriteRepository.ChangePassword(ctx, models.ChangePasswordParams{
		ID:           credentials.ID,
		PasswordHash: newPasswordHash,
	})
	if err != nil {
		return err
	}

	userevents.DispatchUserPasswordChanged(ctx, d.EventDispatcher, userevents.NewUserPasswordChangedParams{
		UserID: credentials.ID,
	})

	return nil
}

func ensurePasswordReplaceable(storedHash string, params ChangePasswordParams) error {
	currentMatches, err := passwordMatchesHash(password.VerificationParams{
		Password: params.CurrentPassword,
		Hash:     storedHash,
	})
	if err != nil {
		return err
	}
	if !currentMatches {
		return e.ErrCurrentPasswordIncorrect{}
	}

	if params.NewPassword == params.CurrentPassword {
		return e.ErrPasswordUnchanged{}
	}

	return nil
}
