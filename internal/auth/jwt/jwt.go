// Package jwt issues and parses the access tokens the API hands to clients.
package jwt

import (
	"errors"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AccessTokenLifetime is how long an issued access token stays valid.
const AccessTokenLifetime = time.Hour

// AccessTokenIssuer is the issuing and parsing port the API layer depends on.
type AccessTokenIssuer interface {
	IssueAccessToken(params IssueAccessTokenParams) (string, error)
	ParseAccessToken(accessToken string) (AccessTokenClaims, error)
}

// NewAccessTokenIssuerParams carries the secret and lifetime NewAccessTokenIssuer signs with.
type NewAccessTokenIssuerParams struct {
	Secret   string
	Duration time.Duration
}

// Issuer is the HS256 AccessTokenIssuer implementation, built by NewAccessTokenIssuer.
type Issuer struct {
	secret   []byte
	duration time.Duration
}

// NewAccessTokenIssuer builds an Issuer, defaulting a zero Duration to AccessTokenLifetime.
func NewAccessTokenIssuer(params NewAccessTokenIssuerParams) (Issuer, error) {
	if params.Secret == "" {
		return Issuer{}, ErrEmptySecret{}
	}

	duration := params.Duration
	if duration == 0 {
		duration = AccessTokenLifetime
	}

	return Issuer{secret: []byte(params.Secret), duration: duration}, nil
}

// IssueAccessTokenParams is the user identity IssueAccessToken writes into a token.
type IssueAccessTokenParams struct {
	UserID uuid.UUID
	Email  string
}

// IssueAccessToken returns a signed access token carrying the user's id and email.
func (i Issuer) IssueAccessToken(params IssueAccessTokenParams) (string, error) {
	issuedAt := time.Now()

	claims := accessTokenClaims{
		Email: params.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   params.UserID.String(),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(issuedAt.Add(i.duration)),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
	if err != nil {
		return "", ErrTokenSigningFailed{Cause: err}
	}

	return accessToken, nil
}

// AccessTokenClaims is the caller-facing form of a parsed token, with its claims already typed.
type AccessTokenClaims struct {
	UserID    uuid.UUID
	Email     string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// ParseAccessToken verifies an access token and returns its claims, or why it was rejected.
func (i Issuer) ParseAccessToken(accessToken string) (AccessTokenClaims, error) {
	parsedClaims := accessTokenClaims{}

	_, err := jwt.ParseWithClaims(accessToken, &parsedClaims, i.resolveSigningKey,
		jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return AccessTokenClaims{}, describeParseFailure(parsedClaims, err)
	}

	return parsedClaims.toAccessTokenClaims()
}

func (i Issuer) resolveSigningKey(*jwt.Token) (any, error) {
	return i.secret, nil
}

type accessTokenClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func (c accessTokenClaims) toAccessTokenClaims() (AccessTokenClaims, error) {
	userID, err := uuid.Parse(c.Subject)
	if err != nil {
		return AccessTokenClaims{}, ErrClaimUnusable{Claim: "sub", Cause: err}
	}
	if c.IssuedAt == nil {
		return AccessTokenClaims{}, ErrClaimUnusable{Claim: "iat", Cause: errClaimAbsent}
	}
	if c.ExpiresAt == nil {
		return AccessTokenClaims{}, ErrClaimUnusable{Claim: "exp", Cause: errClaimAbsent}
	}

	return AccessTokenClaims{
		UserID:    userID,
		Email:     c.Email,
		IssuedAt:  c.IssuedAt.Time,
		ExpiresAt: c.ExpiresAt.Time,
	}, nil
}

func describeParseFailure(parsedClaims accessTokenClaims, err error) error {
	if !errors.Is(err, jwt.ErrTokenExpired) {
		return ErrTokenUnparsable{Cause: err}
	}

	expiresAt := time.Time{}
	if parsedClaims.ExpiresAt != nil {
		expiresAt = parsedClaims.ExpiresAt.Time
	}

	return ErrTokenExpired{ExpiresAt: expiresAt}
}
