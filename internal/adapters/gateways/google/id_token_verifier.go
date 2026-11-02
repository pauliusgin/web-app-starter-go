// Package google verifies Sign in with Google ID tokens against google's published signing certificates.
package google

import (
	"context"
	"errors"
	"net/url"
	"slices"

	"google.golang.org/api/idtoken"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/googleidtoken"
)

var _ googleidtoken.Verifier = IDTokenVerifier{}

// Google issues ID tokens under either spelling; idtoken.Validate checks neither, so the verifier does.
var googleIssuers = []string{"accounts.google.com", "https://accounts.google.com"}

// IDTokenVerifier checks ID tokens issued for one OAuth client, caching google's certificates between calls.
type IDTokenVerifier struct {
	validator *idtoken.Validator
	clientID  string
}

// NewIDTokenVerifierParams is the OAuth client ID the ID tokens must be issued for.
type NewIDTokenVerifierParams struct {
	ClientID string
}

// NewIDTokenVerifier builds the verifier the process reuses; an empty client ID is refused, since the
// validator would then skip the audience check and accept tokens issued for any app.
func NewIDTokenVerifier(ctx context.Context, params NewIDTokenVerifierParams) (IDTokenVerifier, error) {
	if params.ClientID == "" {
		return IDTokenVerifier{}, ErrClientIDMissing{}
	}

	validator, err := idtoken.NewValidator(ctx)
	if err != nil {
		return IDTokenVerifier{}, ErrValidatorCreationFailed{Cause: err}
	}

	return IDTokenVerifier{validator: validator, clientID: params.ClientID}, nil
}

// VerifyIDToken checks the token's signature, audience, expiry and issuer and answers the identity it proves.
func (v IDTokenVerifier) VerifyIDToken(ctx context.Context, idToken string) (googleidtoken.Identity, error) {
	payload, err := v.validator.Validate(ctx, idToken, v.clientID)
	if errors.As(err, new(*url.Error)) {
		return googleidtoken.Identity{}, googleidtoken.ErrIDTokenVerificationFailed{Cause: err}
	}
	if err != nil {
		return googleidtoken.Identity{}, googleidtoken.ErrIDTokenInvalid{Cause: err}
	}

	return identityFromPayload(payload)
}

func identityFromPayload(payload *idtoken.Payload) (googleidtoken.Identity, error) {
	if !slices.Contains(googleIssuers, payload.Issuer) {
		return googleidtoken.Identity{}, googleidtoken.ErrIDTokenInvalid{Cause: ErrIssuerUnknown{Issuer: payload.Issuer}}
	}

	email, _ := payload.Claims["email"].(string)
	if payload.Subject == "" || email == "" {
		return googleidtoken.Identity{}, googleidtoken.ErrIDTokenInvalid{Cause: ErrIdentityClaimsMissing{}}
	}

	return googleidtoken.Identity{
		Subject:       payload.Subject,
		Email:         email,
		EmailVerified: emailVerifiedClaim(payload.Claims["email_verified"]),
	}, nil
}

// Google has sent email_verified both as a JSON boolean and as the string "true".
func emailVerifiedClaim(claim any) bool {
	return claim == true || claim == "true"
}
