package googleidtoken

import "fmt"

const idTokenInvalidText = "expected a google ID token that is signed by google, unexpired and issued for this app"

const idTokenVerificationFailedText = "failed reaching google to verify the ID token"

// ErrIDTokenInvalid reports an ID token that is malformed, forged, expired or issued for another app.
type ErrIDTokenInvalid struct {
	Cause error
}

func (e ErrIDTokenInvalid) Error() string {
	return fmt.Sprintf("invalid google ID token: %s: %s", e.Cause, idTokenInvalidText)
}

func (e ErrIDTokenInvalid) Unwrap() error {
	return e.Cause
}

// ErrIDTokenVerificationFailed reports a verification that could not fetch google's signing certificates.
type ErrIDTokenVerificationFailed struct {
	Cause error
}

func (e ErrIDTokenVerificationFailed) Error() string {
	return fmt.Sprintf("%s: %s", idTokenVerificationFailedText, e.Cause)
}

func (e ErrIDTokenVerificationFailed) Unwrap() error {
	return e.Cause
}
