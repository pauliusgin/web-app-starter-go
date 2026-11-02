package googleidtoken

import (
	"context"
	"errors"
)

var _ Verifier = InMemoryVerifier{}

var errUnknownIDToken = errors.New("ID token is not one the in-memory verifier was given")

// InMemoryVerifier is a Verifier answering the identities it was given, keyed by the ID token.
type InMemoryVerifier struct {
	Identities map[string]Identity

	// VerifyFailure is returned for every token when set, standing in for google being unreachable.
	VerifyFailure error
}

// VerifyIDToken answers the identity stored for idToken, or ErrIDTokenInvalid for an unknown token.
func (v InMemoryVerifier) VerifyIDToken(_ context.Context, idToken string) (Identity, error) {
	if v.VerifyFailure != nil {
		return Identity{}, v.VerifyFailure
	}

	identity, found := v.Identities[idToken]
	if !found {
		return Identity{}, ErrIDTokenInvalid{Cause: errUnknownIDToken}
	}

	return identity, nil
}
