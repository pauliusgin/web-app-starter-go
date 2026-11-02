package google

import "fmt"

const clientIDMissingText = "expected the google OAuth client ID ID tokens must be issued for"

const validatorCreationFailedText = "failed building the google ID token validator"

const issuerUnknownText = "expected an ID token issued by accounts.google.com"

const identityClaimsMissingText = "expected an ID token carrying the sub and email claims"

// ErrClientIDMissing reports a verifier built without the client ID the audience check needs.
type ErrClientIDMissing struct{}

func (e ErrClientIDMissing) Error() string {
	return clientIDMissingText
}

// ErrValidatorCreationFailed reports an idtoken validator whose HTTP client could not be built.
type ErrValidatorCreationFailed struct {
	Cause error
}

func (e ErrValidatorCreationFailed) Error() string {
	return fmt.Sprintf("%s: %s", validatorCreationFailedText, e.Cause)
}

func (e ErrValidatorCreationFailed) Unwrap() error {
	return e.Cause
}

// ErrIssuerUnknown reports a correctly signed token whose iss claim is not google's.
type ErrIssuerUnknown struct {
	Issuer string
}

func (e ErrIssuerUnknown) Error() string {
	return fmt.Sprintf("unknown ID token issuer %q: %s", e.Issuer, issuerUnknownText)
}

// ErrIdentityClaimsMissing reports a token lacking the sub or email claim the account is matched on.
type ErrIdentityClaimsMissing struct{}

func (e ErrIdentityClaimsMissing) Error() string {
	return identityClaimsMissingText
}
