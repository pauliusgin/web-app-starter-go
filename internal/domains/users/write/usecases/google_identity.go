package usecases

import (
	"context"
	"strings"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/googleidtoken"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
)

// VerifyGoogleIDToken is the google ID token check signing up or in with google depends on.
type VerifyGoogleIDToken func(ctx context.Context, idToken string) (googleidtoken.Identity, error)

type verifyGoogleIdentityParams struct {
	VerifyGoogleIDToken VerifyGoogleIDToken
	IDToken             string
}

// verifyGoogleIdentity answers the token's identity only when google vouches for the email, since the
// account is matched and created on it without sending a verification link of our own.
func verifyGoogleIdentity(ctx context.Context, params verifyGoogleIdentityParams) (googleidtoken.Identity, error) {
	identity, err := params.VerifyGoogleIDToken(ctx, params.IDToken)
	if err != nil {
		return googleidtoken.Identity{}, err
	}

	// Stored emails went through the HTTP validator's lowercase+trim; google's must match them the same way.
	identity.Email = strings.ToLower(strings.TrimSpace(identity.Email))
	if !identity.EmailVerified {
		return googleidtoken.Identity{}, e.ErrGoogleEmailUnverified{Email: identity.Email}
	}

	return identity, nil
}
