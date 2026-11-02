package usecases

import (
	"context"
	"errors"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	userevents "github.com/pauliusgin/web_app_starter-be/internal/events/user"
)

// SignUpUserGoogleParams is the Sign in with Google ID token the frontend received.
type SignUpUserGoogleParams struct {
	IDToken string
}

// SignUpUserGoogleResult is the google user the signup stored.
type SignUpUserGoogleResult struct {
	User models.User
}

// SignUpUserGoogleDependencies carries the user store, the event dispatcher and the google ID token check.
type SignUpUserGoogleDependencies struct {
	UserWriteRepository repositories.UserRepository
	EventDispatcher     events.Dispatcher
	VerifyGoogleIDToken VerifyGoogleIDToken
}

// SignUpUserGoogle stores a verified, password-less user for a new google account and announces the signup.
func SignUpUserGoogle(
	ctx context.Context,
	d SignUpUserGoogleDependencies,
	params SignUpUserGoogleParams,
) (SignUpUserGoogleResult, error) {
	identity, err := verifyGoogleIdentity(ctx, verifyGoogleIdentityParams{
		VerifyGoogleIDToken: d.VerifyGoogleIDToken,
		IDToken:             params.IDToken,
	})
	if err != nil {
		return SignUpUserGoogleResult{}, err
	}

	if err := ensureGoogleAccountAvailable(ctx, ensureGoogleAccountAvailableParams{
		UserWriteRepository: d.UserWriteRepository,
		GoogleSubject:       identity.Subject,
	}); err != nil {
		return SignUpUserGoogleResult{}, err
	}

	if err := ensureEmailAvailable(ctx, ensureEmailAvailableParams{
		UserWriteRepository: d.UserWriteRepository,
		Email:               identity.Email,
	}); err != nil {
		return SignUpUserGoogleResult{}, err
	}

	newUser, err := d.UserWriteRepository.SaveGoogleUser(ctx, models.NewGoogle(models.NewGoogleUserParams{
		Email:         identity.Email,
		GoogleSubject: identity.Subject,
	}))
	if err != nil {
		return SignUpUserGoogleResult{}, err
	}

	userevents.DispatchUserSignedUp(ctx, d.EventDispatcher, userevents.NewUserSignedUpParams{
		UserID: newUser.ID,
		Email:  newUser.Email,
	})

	return SignUpUserGoogleResult{User: newUser}, nil
}

type ensureGoogleAccountAvailableParams struct {
	UserWriteRepository repositories.UserRepository
	GoogleSubject       string
}

func ensureGoogleAccountAvailable(ctx context.Context, params ensureGoogleAccountAvailableParams) error {
	_, err := params.UserWriteRepository.FindUserByGoogleSubject(ctx, params.GoogleSubject)
	if errors.As(err, &e.ErrUserNotFound{}) {
		return nil
	}
	if err != nil {
		return err
	}

	return e.ErrGoogleAccountTaken{GoogleSubject: params.GoogleSubject}
}
