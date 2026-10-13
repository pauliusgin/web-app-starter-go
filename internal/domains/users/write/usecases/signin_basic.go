package usecases

import (
	"context"
	"errors"
	"sync"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/password"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	writerepo "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	userevents "github.com/pauliusgin/web_app_starter-be/internal/events/user"
)

// SignInUserParams is the plaintext sign-in input checked against the stored password hash.
type SignInUserParams struct {
	Email    string
	Password string
}

// SignInUserResult is the user whose credentials matched, stamped as signed in.
type SignInUserResult struct {
	User models.User
}

// SignInUserDependencies carries the collaborators signing a user in needs.
type SignInUserDependencies struct {
	UserWriteRepository writerepo.UserRepository
	EventDispatcher     events.Dispatcher
}

var unknownAccountDecoyHash = sync.OnceValues(func() (string, error) {
	return password.Hash("unknown-account-decoy-password")
})

// SignInUser checks the email and password, then stamps and announces the sign-in on a match.
func SignInUser(
	ctx context.Context,
	d SignInUserDependencies,
	params SignInUserParams,
) (SignInUserResult, error) {
	credentials, err := d.UserWriteRepository.FindUserCredentialsByEmail(ctx, params.Email)
	if errors.As(err, &e.ErrUserNotFound{}) {
		return SignInUserResult{}, rejectUnknownAccount(params.Password)
	}
	if err != nil {
		return SignInUserResult{}, err
	}

	err = verifyUserPassword(password.VerificationParams{
		Password: params.Password,
		Hash:     credentials.PasswordHash,
	})
	if err != nil {
		return SignInUserResult{}, err
	}

	signedInUser, err := d.UserWriteRepository.MarkSignedIn(ctx, credentials.ID)
	if err != nil {
		return SignInUserResult{}, err
	}

	userevents.DispatchUserSignedIn(ctx, d.EventDispatcher, userevents.NewUserSignedInParams{
		UserID: signedInUser.ID,
		Email:  signedInUser.Email,
	})

	return SignInUserResult{User: signedInUser}, nil
}

func verifyUserPassword(params password.VerificationParams) error {
	matches, err := passwordMatchesHash(params)
	if err != nil {
		return err
	}
	if !matches {
		return e.ErrInvalidCredentials{}
	}

	return nil
}

func rejectUnknownAccount(candidatePassword string) error {
	decoyHash, err := unknownAccountDecoyHash()
	if err == nil {
		_ = verifyUserPassword(password.VerificationParams{Password: candidatePassword, Hash: decoyHash})
	}

	return e.ErrInvalidCredentials{}
}

func passwordMatchesHash(params password.VerificationParams) (bool, error) {
	matches, err := password.Verify(params)
	if err != nil {
		return false, e.ErrPasswordVerificationFailed{Cause: err}
	}

	return matches, nil
}
