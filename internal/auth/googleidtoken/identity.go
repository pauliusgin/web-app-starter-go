// Package googleidtoken declares the verified google identity a Sign in with Google ID token proves.
package googleidtoken

import "context"

// Identity is the google account a verified ID token was issued for.
type Identity struct {
	// Subject is google's stable account id (the `sub` claim); unlike Email it never changes.
	Subject       string
	Email         string
	EmailVerified bool
}

// Verifier checks an ID token's signature, issuer, audience and expiry, answering the identity it proves.
type Verifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (Identity, error)
}
