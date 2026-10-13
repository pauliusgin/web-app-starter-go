package middleware

import (
	"fmt"

	"github.com/google/uuid"
)

const authorizationHeaderExpectationText = "expected an Authorization header"
const bearerSchemeExpectationText = "expected the Bearer scheme"
const bearerTokenExpectationText = "expected \"Bearer <token>\": one space, and a token holding no whitespace"
const accessTokenRejectedText = "the access token was rejected"
const authenticatedUserNotFoundText = "the access token names a user that no longer exists"

// ErrAuthorizationHeaderAbsent reports a request that carried no Authorization header.
type ErrAuthorizationHeaderAbsent struct{}

func (e ErrAuthorizationHeaderAbsent) Error() string {
	return fmt.Sprintf("missing Authorization header: %s", authorizationHeaderExpectationText)
}

// ErrUnsupportedAuthorizationScheme reports an Authorization header in a scheme other than Bearer.
type ErrUnsupportedAuthorizationScheme struct {
	// Scheme is truncated, since a bare token in the header arrives here as the scheme.
	Scheme string
}

func (e ErrUnsupportedAuthorizationScheme) Error() string {
	return fmt.Sprintf("unsupported authorization scheme %q: %s", e.Scheme, bearerSchemeExpectationText)
}

// ErrBearerTokenMalformed reports a Bearer header whose token is absent, empty, or badly separated.
type ErrBearerTokenMalformed struct{}

// Error deliberately echoes nothing back: the rest of the header is token material, and a
// malformed token is still a secret a client may retry with a working one.
func (e ErrBearerTokenMalformed) Error() string {
	return fmt.Sprintf("malformed bearer credentials: %s", bearerTokenExpectationText)
}

// ErrAccessTokenRejected reports an access token the jwt package refused to accept.
type ErrAccessTokenRejected struct {
	Cause error
}

func (e ErrAccessTokenRejected) Error() string {
	return fmt.Sprintf("%s: %s", accessTokenRejectedText, e.Cause)
}

func (e ErrAccessTokenRejected) Unwrap() error {
	return e.Cause
}

// ErrAuthenticatedUserNotFound reports a valid access token whose user has since been deleted.
type ErrAuthenticatedUserNotFound struct {
	UserID uuid.UUID
	Cause  error
}

func (e ErrAuthenticatedUserNotFound) Error() string {
	return fmt.Sprintf("user %s is gone: %s: %s", e.UserID, authenticatedUserNotFoundText, e.Cause)
}

func (e ErrAuthenticatedUserNotFound) Unwrap() error {
	return e.Cause
}
