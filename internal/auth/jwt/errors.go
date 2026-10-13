package jwt

import (
	"errors"
	"fmt"
	"time"
)

const emptySecretExpectationText = "expected a non-empty jwt secret"
const tokenSigningFailedText = "failed signing the access token"
const tokenUnparsableText = "expected a token signed with HS256 by this issuer's secret"
const claimUnusableText = "expected a claim this package issues"

var errClaimAbsent = errors.New("claim is absent")

// ErrEmptySecret reports an issuer built without a signing secret.
type ErrEmptySecret struct{}

func (e ErrEmptySecret) Error() string {
	return fmt.Sprintf("missing jwt secret: %s", emptySecretExpectationText)
}

// ErrTokenSigningFailed reports an access token the signer refused to produce.
type ErrTokenSigningFailed struct {
	Cause error
}

func (e ErrTokenSigningFailed) Error() string {
	return fmt.Sprintf("%s: %s", tokenSigningFailedText, e.Cause)
}

func (e ErrTokenSigningFailed) Unwrap() error {
	return e.Cause
}

// ErrTokenUnparsable reports a token this issuer's secret and method could not open.
type ErrTokenUnparsable struct {
	Cause error
}

func (e ErrTokenUnparsable) Error() string {
	return fmt.Sprintf("unparsable access token: %s: %s", e.Cause, tokenUnparsableText)
}

func (e ErrTokenUnparsable) Unwrap() error {
	return e.Cause
}

// ErrTokenExpired reports a well-formed token whose expiry has already passed.
type ErrTokenExpired struct {
	ExpiresAt time.Time
}

func (e ErrTokenExpired) Error() string {
	return fmt.Sprintf("access token expired at %s", e.ExpiresAt.Format(time.RFC3339))
}

// ErrClaimUnusable reports a claim that is absent or holds a value this package cannot use.
type ErrClaimUnusable struct {
	Claim string
	Cause error
}

func (e ErrClaimUnusable) Error() string {
	return fmt.Sprintf("unusable %s claim: %s: %s", e.Claim, e.Cause, claimUnusableText)
}

func (e ErrClaimUnusable) Unwrap() error {
	return e.Cause
}
