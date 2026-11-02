package usecases

import (
	"context"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/googleidtoken"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	userevents "github.com/pauliusgin/web_app_starter-be/internal/events/user"
)

// AddGoogleAccountParams is the authenticated password user and the Sign in with Google ID token they connect.
type AddGoogleAccountParams struct {
	UserID  uuid.UUID
	IDToken string
}

// AddGoogleAccountDependencies carries the user store, the event dispatcher and the google ID token check.
type AddGoogleAccountDependencies struct {
	UserWriteRepository repositories.UserRepository
	EventDispatcher     events.Dispatcher
	VerifyGoogleIDToken VerifyGoogleIDToken
}

// AddGoogleAccount connects the google account holding the user's email, so it signs in with either, then announces it.
func AddGoogleAccount(ctx context.Context, d AddGoogleAccountDependencies, params AddGoogleAccountParams) error {
	credentials, err := d.UserWriteRepository.FindUserCredentialsByID(ctx, params.UserID)
	if err != nil {
		return err
	}
	if credentials.HasGoogleAccount() {
		return e.ErrGoogleAccountAlreadyConnected{UserID: credentials.ID}
	}

	identity, err := verifyAccountGoogleIdentity(ctx, verifyAccountGoogleIdentityParams{
		VerifyGoogleIDToken: d.VerifyGoogleIDToken,
		IDToken:             params.IDToken,
		Credentials:         credentials,
	})
	if err != nil {
		return err
	}

	if err := ensureGoogleAccountAvailable(ctx, ensureGoogleAccountAvailableParams{
		UserWriteRepository: d.UserWriteRepository,
		GoogleSubject:       identity.Subject,
	}); err != nil {
		return err
	}

	err = d.UserWriteRepository.AddGoogleAccount(ctx, models.AddGoogleAccountParams{
		ID:            credentials.ID,
		GoogleSubject: identity.Subject,
	})
	if err != nil {
		return err
	}

	userevents.DispatchUserGoogleAccountAdded(ctx, d.EventDispatcher, userevents.NewUserGoogleAccountAddedParams{
		UserID: credentials.ID,
	})

	return nil
}

type verifyAccountGoogleIdentityParams struct {
	VerifyGoogleIDToken VerifyGoogleIDToken
	IDToken             string
	Credentials         models.UserCredentials
}

// verifyAccountGoogleIdentity accepts only a google account holding the account's own email: google's verified
// email is the proof the same person owns both, so connecting a differently-addressed google account is refused.
func verifyAccountGoogleIdentity(
	ctx context.Context,
	params verifyAccountGoogleIdentityParams,
) (googleidtoken.Identity, error) {
	identity, err := verifyGoogleIdentity(ctx, verifyGoogleIdentityParams{
		VerifyGoogleIDToken: params.VerifyGoogleIDToken,
		IDToken:             params.IDToken,
	})
	if err != nil {
		return googleidtoken.Identity{}, err
	}
	if identity.Email != params.Credentials.Email {
		return googleidtoken.Identity{}, e.ErrGoogleEmailMismatch{
			UserID:      params.Credentials.ID,
			GoogleEmail: identity.Email,
		}
	}

	return identity, nil
}
