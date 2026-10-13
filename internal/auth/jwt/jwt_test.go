package jwt

import (
	"errors"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "a-test-signing-secret-that-is-long-enough"

const testEmail = "accountant@example.com"

func newIssuerForTest(t *testing.T, tokenLifetime time.Duration) AccessTokenIssuer {
	t.Helper()

	issuer, err := NewAccessTokenIssuer(NewAccessTokenIssuerParams{
		Secret:   testSecret,
		Duration: tokenLifetime,
	})
	if err != nil {
		t.Fatalf("NewAccessTokenIssuer: %v", err)
	}

	return issuer
}

func issueTokenForTest(t *testing.T, issuer AccessTokenIssuer, userID uuid.UUID) string {
	t.Helper()

	rawToken, err := issuer.IssueAccessToken(IssueAccessTokenParams{UserID: userID, Email: testEmail})
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}

	return rawToken
}

func signClaimsForTest(t *testing.T, method jwtlib.SigningMethod, claims jwtlib.Claims, key any) string {
	t.Helper()

	rawToken, err := jwtlib.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatalf("signing a token by hand: %v", err)
	}

	return rawToken
}

func TestIssueAccessTokenRoundTripsTheIdentity(t *testing.T) {
	issuer := newIssuerForTest(t, 0)
	userID := uuid.New()

	claims, err := issuer.ParseAccessToken(issueTokenForTest(t, issuer, userID))
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("UserID = %s, want %s", claims.UserID, userID)
	}
	if claims.Email != testEmail {
		t.Errorf("Email = %q, want %q", claims.Email, testEmail)
	}
	if claims.IssuedAt.IsZero() {
		t.Error("IssuedAt is the zero time, want the moment the token was issued")
	}
	if !claims.ExpiresAt.After(claims.IssuedAt) {
		t.Errorf("ExpiresAt = %s, want it after IssuedAt = %s", claims.ExpiresAt, claims.IssuedAt)
	}
}

func TestIssuedTokensCarryTheConfiguredLifetime(t *testing.T) {
	cases := []struct {
		name          string
		tokenLifetime time.Duration
		wantLifetime  time.Duration
	}{
		{name: "zero falls back to the default", tokenLifetime: 0, wantLifetime: AccessTokenLifetime},
		{name: "shorter than the default", tokenLifetime: 15 * time.Minute, wantLifetime: 15 * time.Minute},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			issuer := newIssuerForTest(t, testCase.tokenLifetime)

			claims, err := issuer.ParseAccessToken(issueTokenForTest(t, issuer, uuid.New()))
			if err != nil {
				t.Fatalf("ParseAccessToken: %v", err)
			}

			gotLifetime := claims.ExpiresAt.Sub(claims.IssuedAt)
			if gotLifetime != testCase.wantLifetime {
				t.Errorf("ExpiresAt - IssuedAt = %s, want %s", gotLifetime, testCase.wantLifetime)
			}
		})
	}
}

func TestNewAccessTokenIssuerRejectsAnEmptySecret(t *testing.T) {
	cases := []struct {
		name   string
		params NewAccessTokenIssuerParams
	}{
		{name: "secret absent", params: NewAccessTokenIssuerParams{}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := NewAccessTokenIssuer(testCase.params)

			emptySecret := ErrEmptySecret{}
			if !errors.As(err, &emptySecret) {
				t.Fatalf("error = %v, want ErrEmptySecret", err)
			}
		})
	}
}

func TestParseAccessTokenRejectsInvalidTokens(t *testing.T) {
	validClaims := jwtlib.MapClaims{
		"sub":   uuid.New().String(),
		"email": testEmail,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(time.Hour).Unix(),
	}

	cases := []struct {
		name     string
		rawToken func(t *testing.T) string
	}{
		{
			name:     "invalid token",
			rawToken: func(*testing.T) string { return "i.am.not.a.token" },
		},
		{
			name: "signed with another secret",
			rawToken: func(t *testing.T) string {
				return signClaimsForTest(t, jwtlib.SigningMethodHS256, validClaims, []byte("a-different-secret"))
			},
		},
		{
			name: "another hmac algorithm",
			rawToken: func(t *testing.T) string {
				return signClaimsForTest(t, jwtlib.SigningMethodHS512, validClaims, []byte(testSecret))
			},
		},
	}

	issuer := newIssuerForTest(t, 0)

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			claims, err := issuer.ParseAccessToken(testCase.rawToken(t))

			unparsable := ErrTokenUnparsable{}
			if !errors.As(err, &unparsable) {
				t.Fatalf("error = %v, want ErrTokenUnparsable", err)
			}
			if unparsable.Cause == nil {
				t.Error("Cause = nil, want the parser error")
			}
			if claims != (AccessTokenClaims{}) {
				t.Errorf("claims = %+v, want the zero value on failure", claims)
			}
		})
	}
}

func TestParseAccessTokenRejectsAnExpiredToken(t *testing.T) {
	issuer := newIssuerForTest(t, -time.Minute)
	rawToken := issueTokenForTest(t, issuer, uuid.New())

	_, err := issuer.ParseAccessToken(rawToken)

	expired := ErrTokenExpired{}
	if !errors.As(err, &expired) {
		t.Fatalf("error = %v, want ErrTokenExpired", err)
	}
	if expired.ExpiresAt.After(time.Now()) {
		t.Errorf("ExpiresAt = %s, want a moment already past", expired.ExpiresAt)
	}
	if errors.As(err, &ErrTokenUnparsable{}) {
		t.Error("an expired token reports as unparsable, want the two failures kept apart")
	}
}

func TestParseAccessTokenRejectsUnusableClaims(t *testing.T) {
	issuer := newIssuerForTest(t, 0)
	issuedAt := time.Now()

	cases := []struct {
		name      string
		claims    jwtlib.MapClaims
		wantClaim string
	}{
		{
			name: "subject is not a uuid",
			claims: jwtlib.MapClaims{
				"sub":   "not-a-uuid",
				"email": testEmail,
				"iat":   issuedAt.Unix(),
				"exp":   issuedAt.Add(time.Hour).Unix(),
			},
			wantClaim: "sub",
		},
		{
			name: "subject absent",
			claims: jwtlib.MapClaims{
				"email": testEmail,
				"iat":   issuedAt.Unix(),
				"exp":   issuedAt.Add(time.Hour).Unix(),
			},
			wantClaim: "sub",
		},
		{
			name: "issued at absent",
			claims: jwtlib.MapClaims{
				"sub":   uuid.New().String(),
				"email": testEmail,
				"exp":   issuedAt.Add(time.Hour).Unix(),
			},
			wantClaim: "iat",
		},
		{
			name: "expiry absent",
			claims: jwtlib.MapClaims{
				"sub":   uuid.New().String(),
				"email": testEmail,
				"iat":   issuedAt.Unix(),
			},
			wantClaim: "exp",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			rawToken := signClaimsForTest(t, jwtlib.SigningMethodHS256, testCase.claims, []byte(testSecret))

			claims, err := issuer.ParseAccessToken(rawToken)

			unusable := ErrClaimUnusable{}
			if !errors.As(err, &unusable) {
				t.Fatalf("error = %v, want ErrClaimUnusable", err)
			}
			if unusable.Claim != testCase.wantClaim {
				t.Errorf("Claim = %q, want %q", unusable.Claim, testCase.wantClaim)
			}
			if claims != (AccessTokenClaims{}) {
				t.Errorf("claims = %+v, want the zero value on failure", claims)
			}
		})
	}
}
