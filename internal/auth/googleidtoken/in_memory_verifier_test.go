package googleidtoken

import (
	"context"
	"errors"
	"testing"
)

func TestInMemoryVerifierAnswersOnlyTheIdentitiesItWasGiven(t *testing.T) {
	stored := Identity{Subject: "109876543210987654321", Email: "accountant@example.com", EmailVerified: true}
	verifier := InMemoryVerifier{Identities: map[string]Identity{"known-token": stored}}

	identity, err := verifier.VerifyIDToken(context.Background(), "known-token")
	if err != nil {
		t.Fatalf("VerifyIDToken(known): %v", err)
	}
	if identity != stored {
		t.Errorf("identity = %+v, want %+v", identity, stored)
	}

	_, err = verifier.VerifyIDToken(context.Background(), "unknown-token")
	if !errors.As(err, &ErrIDTokenInvalid{}) {
		t.Errorf("VerifyIDToken(unknown) err = %v, want ErrIDTokenInvalid", err)
	}
}
