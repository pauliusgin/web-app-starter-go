package google

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/api/idtoken"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/googleidtoken"
)

func TestIdentityFromPayloadMapsTheGoogleClaims(t *testing.T) {
	cases := []struct {
		name              string
		issuer            string
		emailVerified     any
		wantEmailVerified bool
	}{
		{name: "bare issuer, boolean verified", issuer: "accounts.google.com", emailVerified: true, wantEmailVerified: true},
		{name: "https issuer, string verified", issuer: "https://accounts.google.com", emailVerified: "true", wantEmailVerified: true},
		{name: "boolean unverified", issuer: "accounts.google.com", emailVerified: false},
		{name: "string unverified", issuer: "accounts.google.com", emailVerified: "false"},
		{name: "verified claim missing", issuer: "accounts.google.com", emailVerified: nil},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			identity, err := identityFromPayload(&idtoken.Payload{
				Issuer:  testCase.issuer,
				Subject: "109876543210987654321",
				Claims:  map[string]any{"email": "accountant@example.com", "email_verified": testCase.emailVerified},
			})
			if err != nil {
				t.Fatalf("identityFromPayload: %v", err)
			}

			want := googleidtoken.Identity{
				Subject:       "109876543210987654321",
				Email:         "accountant@example.com",
				EmailVerified: testCase.wantEmailVerified,
			}
			if identity != want {
				t.Errorf("identity = %+v, want %+v", identity, want)
			}
		})
	}
}

func TestIdentityFromPayloadRejectsTokensItCannotTrust(t *testing.T) {
	cases := []struct {
		name     string
		payload  idtoken.Payload
		hasCause func(err error) bool
	}{
		{
			name:     "foreign issuer",
			payload:  idtoken.Payload{Issuer: "https://evil.example.com", Subject: "1", Claims: map[string]any{"email": "a@example.com"}},
			hasCause: isIssuerUnknown,
		},
		{
			name:     "subject missing",
			payload:  idtoken.Payload{Issuer: "accounts.google.com", Claims: map[string]any{"email": "a@example.com"}},
			hasCause: isIdentityClaimsMissing,
		},
		{
			name:     "email missing",
			payload:  idtoken.Payload{Issuer: "accounts.google.com", Subject: "1", Claims: map[string]any{}},
			hasCause: isIdentityClaimsMissing,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			identity, err := identityFromPayload(&testCase.payload)

			if !errors.As(err, &googleidtoken.ErrIDTokenInvalid{}) {
				t.Fatalf("err = %v, want ErrIDTokenInvalid", err)
			}
			if !testCase.hasCause(err) {
				t.Errorf("err = %v, want the %s cause", err, testCase.name)
			}
			if identity != (googleidtoken.Identity{}) {
				t.Errorf("identity = %+v, want zero", identity)
			}
		})
	}
}

func isIssuerUnknown(err error) bool {
	return errors.As(err, &ErrIssuerUnknown{})
}

func isIdentityClaimsMissing(err error) bool {
	return errors.As(err, &ErrIdentityClaimsMissing{})
}

func TestNewIDTokenVerifierRefusesAnEmptyClientID(t *testing.T) {
	_, err := NewIDTokenVerifier(context.Background(), NewIDTokenVerifierParams{})

	if !errors.As(err, &ErrClientIDMissing{}) {
		t.Errorf("err = %v, want ErrClientIDMissing", err)
	}
}

func TestVerifyIDTokenRejectsAMalformedToken(t *testing.T) {
	verifier, err := NewIDTokenVerifier(context.Background(), NewIDTokenVerifierParams{ClientID: "client.apps.googleusercontent.com"})
	if err != nil {
		t.Fatalf("NewIDTokenVerifier: %v", err)
	}

	_, err = verifier.VerifyIDToken(context.Background(), "not-a-jwt")

	if !errors.As(err, &googleidtoken.ErrIDTokenInvalid{}) {
		t.Errorf("err = %v, want ErrIDTokenInvalid", err)
	}
}
