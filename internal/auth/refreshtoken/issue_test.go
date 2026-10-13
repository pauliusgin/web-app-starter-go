package refreshtoken

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
)

func TestIssueRefreshTokenStoresTheHashAndReturnsThePlaintext(t *testing.T) {
	store := newInMemoryTokenStore()
	userID := uuid.New()

	issuedAfter := time.Now()
	issued, err := Issue(context.Background(),
		IssuingDependencies{WriteRepository: store},
		IssuingParams{UserID: userID})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	issuedBefore := time.Now()

	storedTokens := store.tokensForUser(userID)
	if len(storedTokens) != 1 {
		t.Fatalf("stored rows = %d, want 1", len(storedTokens))
	}
	storedToken := storedTokens[0]

	if issued.Plaintext == "" {
		t.Fatal("Plaintext is empty, want the minted secret")
	}
	if storedToken.TokenHash != token.Hash(issued.Plaintext) {
		t.Errorf("stored hash = %q, want the hash of the returned plaintext %q",
			storedToken.TokenHash, token.Hash(issued.Plaintext))
	}
	if storedToken.TokenHash == issued.Plaintext {
		t.Error("the repository received the plaintext, want only its hash")
	}
	if storedToken.ID == uuid.Nil {
		t.Error("ID is the nil uuid, want a generated one")
	}
	if storedToken.UserID != userID {
		t.Errorf("UserID = %s, want %s", storedToken.UserID, userID)
	}
	if storedToken.IsRevoked() {
		t.Error("a freshly issued token is revoked, want it live")
	}
	if !storedToken.ExpiresAt.Equal(issued.ExpiresAt) {
		t.Errorf("stored ExpiresAt = %s, want the returned %s", storedToken.ExpiresAt, issued.ExpiresAt)
	}
	if issued.ExpiresAt.Before(issuedAfter.Add(Lifetime)) ||
		issued.ExpiresAt.After(issuedBefore.Add(Lifetime)) {
		t.Errorf("ExpiresAt = %s, want one lifetime past the issue between %s and %s",
			issued.ExpiresAt, issuedAfter, issuedBefore)
	}
}

func TestIssueRefreshTokenSurfacesAStorageFailure(t *testing.T) {
	cause := errors.New("pq: connection refused")
	store := newInMemoryTokenStore()
	store.saveFailure = ErrSaveRefreshTokenFailed{Cause: cause}

	issued, err := Issue(context.Background(),
		IssuingDependencies{WriteRepository: store},
		IssuingParams{UserID: uuid.New()})

	if !errors.As(err, &ErrSaveRefreshTokenFailed{}) {
		t.Fatalf("error = %v, want ErrSaveRefreshTokenFailed", err)
	}
	if !errors.Is(err, cause) {
		t.Error("the storage failure does not unwrap to the driver error")
	}
	if issued != (Issued{}) {
		t.Errorf("token = %+v, want the zero value on failure", issued)
	}
	if len(store.rows) != 0 {
		t.Errorf("stored rows = %d, want nothing stored when the insert fails", len(store.rows))
	}
}
