package usecases

import (
	"context"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/password"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	userevents "github.com/pauliusgin/web_app_starter-be/internal/events/user"
)

// AddPasswordParams is the authenticated google-only user and the password they want to sign in with too.
type AddPasswordParams struct {
	UserID   uuid.UUID
	Password string
}

// AddPasswordDependencies carries the user store holding the credentials and the dispatcher announcing the addition.
type AddPasswordDependencies struct {
	UserWriteRepository repositories.UserRepository
	EventDispatcher     events.Dispatcher
}

// AddPassword stores a password hash on a google-only account, so it signs in with either, then announces it.
func AddPassword(ctx context.Context, d AddPasswordDependencies, params AddPasswordParams) error {
	credentials, err := d.UserWriteRepository.FindUserCredentialsByID(ctx, params.UserID)
	if err != nil {
		return err
	}
	if credentials.HasPassword() {
		return e.ErrPasswordAlreadySet{UserID: credentials.ID}
	}
	if !credentials.HasGoogleAccount() {
		return e.ErrGoogleAccountNotConnected{UserID: credentials.ID}
	}

	passwordHash, err := password.Hash(params.Password)
	if err != nil {
		return e.ErrPasswordHashingFailed{Cause: err}
	}

	err = d.UserWriteRepository.AddPassword(ctx, models.AddPasswordParams{
		ID:           credentials.ID,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return err
	}

	userevents.DispatchUserPasswordAdded(ctx, d.EventDispatcher, userevents.NewUserPasswordAddedParams{
		UserID: credentials.ID,
	})

	return nil
}
