package refreshtoken

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
)

func rotateForTest(
	t *testing.T,
	store *inMemoryTokenStore,
	presentedToken string,
) (Rotated, error) {
	t.Helper()

	return Rotate(context.Background(),
		RotateDependencies{
			ReadRepository:  store,
			WriteRepository: store,
		},
		RotateParams{PresentedToken: presentedToken})
}

func TestRotateRefreshTokenReplacesALiveTokenWithANewOne(t *testing.T) {
	store := newInMemoryTokenStore()
	userID := uuid.New()
	presentedToken := store.seedToken(t, seedTokenParams{UserID: userID})

	rotatedAfter := time.Now()
	rotated, err := rotateForTest(t, store, presentedToken)
	if err != nil {
		t.Fatalf("RotateRefreshToken: %v", err)
	}
	rotatedBefore := time.Now()

	if rotated.UserID != userID {
		t.Errorf("UserID = %s, want %s", rotated.UserID, userID)
	}
	if rotated.Plaintext == presentedToken {
		t.Error("the replacement token is the presented one, want a freshly minted secret")
	}
	if rotated.ExpiresAt.Before(rotatedAfter.Add(Lifetime)) ||
		rotated.ExpiresAt.After(rotatedBefore.Add(Lifetime)) {
		t.Errorf("ExpiresAt = %s, want one lifetime past the rotation between %s and %s",
			rotated.ExpiresAt, rotatedAfter, rotatedBefore)
	}

	if !store.tokenByPlaintext(t, presentedToken).IsRevoked() {
		t.Error("the presented token is still live, want it revoked by the rotation")
	}
	replacementToken := store.tokenByPlaintext(t, rotated.Plaintext)
	if replacementToken.IsRevoked() {
		t.Error("the replacement token is revoked, want it live")
	}
	if replacementToken.UserID != userID {
		t.Errorf("replacement UserID = %s, want %s", replacementToken.UserID, userID)
	}
	if replacementToken.TokenHash != token.Hash(rotated.Plaintext) {
		t.Errorf("stored hash = %q, want the hash of the returned plaintext",
			replacementToken.TokenHash)
	}
	if len(store.tokensForUser(userID)) != 2 {
		t.Errorf("stored rows = %d, want the revoked original beside its replacement",
			len(store.tokensForUser(userID)))
	}
}

func TestRotateRefreshTokenSurfacesALookupFailureUnchanged(t *testing.T) {
	cause := errors.New("pq: connection refused")
	store := newInMemoryTokenStore()
	userID := uuid.New()
	presentedToken := store.seedToken(t, seedTokenParams{UserID: userID})
	store.lookupFailure = ErrRefreshTokenLookupFailed{Cause: cause}

	rotated, err := rotateForTest(t, store, presentedToken)

	if !errors.As(err, &ErrRefreshTokenLookupFailed{}) {
		t.Fatalf("error = %v, want ErrRefreshTokenLookupFailed", err)
	}
	if !errors.Is(err, cause) {
		t.Error("the lookup failure does not unwrap to the driver error")
	}
	if rotated != (Rotated{}) {
		t.Errorf("token = %+v, want the zero value on failure", rotated)
	}
	if store.tokenByPlaintext(t, presentedToken).IsRevoked() {
		t.Error("the presented token was revoked, want it untouched when the lookup failed")
	}
	if len(store.tokensForUser(userID)) != 1 {
		t.Errorf("stored rows = %d, want no replacement issued when the lookup failed",
			len(store.tokensForUser(userID)))
	}
}

func TestRotateRefreshTokenSurfacesARevokeFailureUnchanged(t *testing.T) {
	cause := errors.New("pq: connection refused")
	store := newInMemoryTokenStore()
	userID := uuid.New()
	presentedToken := store.seedToken(t, seedTokenParams{UserID: userID})
	store.revokeFailure = ErrRevokeRefreshTokenFailed{Cause: cause}

	rotated, err := rotateForTest(t, store, presentedToken)

	if !errors.As(err, &ErrRevokeRefreshTokenFailed{}) {
		t.Fatalf("error = %v, want ErrRevokeRefreshTokenFailed", err)
	}
	if !errors.Is(err, cause) {
		t.Error("the revoke failure does not unwrap to the driver error")
	}
	if rotated != (Rotated{}) {
		t.Errorf("token = %+v, want the zero value on failure", rotated)
	}
	if len(store.tokensForUser(userID)) != 1 {
		t.Errorf("stored rows = %d, want no replacement issued when the revoke failed",
			len(store.tokensForUser(userID)))
	}
}

func TestRotateRefreshTokenSurfacesAnIssueFailureUnchanged(t *testing.T) {
	cause := errors.New("pq: connection refused")
	store := newInMemoryTokenStore()
	userID := uuid.New()
	presentedToken := store.seedToken(t, seedTokenParams{UserID: userID})
	store.saveFailure = ErrSaveRefreshTokenFailed{Cause: cause}

	rotated, err := rotateForTest(t, store, presentedToken)

	if !errors.As(err, &ErrSaveRefreshTokenFailed{}) {
		t.Fatalf("error = %v, want ErrSaveRefreshTokenFailed", err)
	}
	if !errors.Is(err, cause) {
		t.Error("the issue failure does not unwrap to the driver error")
	}
	if rotated != (Rotated{}) {
		t.Errorf("token = %+v, want the zero value on failure", rotated)
	}
	if len(store.tokensForUser(userID)) != 1 {
		t.Errorf("stored rows = %d, want no replacement stored when issuing failed",
			len(store.tokensForUser(userID)))
	}
}
