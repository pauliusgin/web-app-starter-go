package usecases

import (
	"context"
	"errors"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/googleidtoken"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	userevents "github.com/pauliusgin/web_app_starter-be/internal/events/user"
)

// SignInUserGoogleParams is the Sign in with Google ID token the frontend received.
type SignInUserGoogleParams struct {
	IDToken string
}

// SignInUserGoogleResult is the user signed up with the google account, stamped as signed in.
type SignInUserGoogleResult struct {
	User models.User
}

// SignInUserGoogleDependencies carries the user store, the event dispatcher and the google ID token check.
type SignInUserGoogleDependencies struct {
	UserWriteRepository repositories.UserRepository
	EventDispatcher     events.Dispatcher
	VerifyGoogleIDToken VerifyGoogleIDToken
}

// SignInUserGoogle matches the google account to its user by google subject, then stamps and announces the sign-in.
func SignInUserGoogle(
	ctx context.Context,
	d SignInUserGoogleDependencies,
	params SignInUserGoogleParams,
) (SignInUserGoogleResult, error) {
	identity, err := verifyGoogleIdentity(ctx, verifyGoogleIdentityParams{
		VerifyGoogleIDToken: d.VerifyGoogleIDToken,
		IDToken:             params.IDToken,
	})
	if err != nil {
		return SignInUserGoogleResult{}, err
	}

	googleUser, err := findGoogleUser(ctx, findGoogleUserParams{
		UserWriteRepository: d.UserWriteRepository,
		Identity:            identity,
	})
	if err != nil {
		return SignInUserGoogleResult{}, err
	}

	signedInUser, err := d.UserWriteRepository.MarkSignedIn(ctx, googleUser.ID)
	if err != nil {
		return SignInUserGoogleResult{}, err
	}

	userevents.DispatchUserSignedIn(ctx, d.EventDispatcher, userevents.NewUserSignedInParams{
		UserID: signedInUser.ID,
		Email:  signedInUser.Email,
	})

	return SignInUserGoogleResult{User: signedInUser}, nil
}

type findGoogleUserParams struct {
	UserWriteRepository repositories.UserRepository
	Identity            googleidtoken.Identity
}

// findGoogleUser matches on the google subject, not the email: the user may have changed their email here,
// and an email match alone would sign a google account into someone's password account.
func findGoogleUser(ctx context.Context, params findGoogleUserParams) (models.User, error) {
	googleUser, err := params.UserWriteRepository.FindUserByGoogleSubject(ctx, params.Identity.Subject)
	if errors.As(err, &e.ErrUserNotFound{}) {
		return models.User{}, rejectUnknownGoogleAccount(ctx, params)
	}
	if err != nil {
		return models.User{}, err
	}

	return googleUser, nil
}

func rejectUnknownGoogleAccount(ctx context.Context, params findGoogleUserParams) error {
	_, err := params.UserWriteRepository.FindUserByEmail(ctx, params.Identity.Email)
	if errors.As(err, &e.ErrUserNotFound{}) {
		return e.ErrGoogleAccountNotFound{GoogleSubject: params.Identity.Subject}
	}
	if err != nil {
		return err
	}

	return e.ErrGoogleAccountNotLinked{Email: params.Identity.Email}
}
