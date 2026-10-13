package refreshtoken

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func revokeForTest(t *testing.T, store *inMemoryTokenStore, presentedToken string) error {
	t.Helper()

	return Revoke(context.Background(),
		RevokeDependencies{ReadRepository: store, WriteRepository: store},
		RevokeParams{PresentedToken: presentedToken})
}

func TestRevokeRefreshTokenRevokesALiveToken(t *testing.T) {
	store := newInMemoryTokenStore()
	userID := uuid.New()
	presentedToken := store.seedToken(t, seedTokenParams{UserID: userID})
	otherSessionToken := store.seedToken(t, seedTokenParams{UserID: userID})

	if err := revokeForTest(t, store, presentedToken); err != nil {
		t.Fatalf("RevokeRefreshToken: %v", err)
	}

	if !store.tokenByPlaintext(t, presentedToken).IsRevoked() {
		t.Error("the presented token is still live, want it revoked")
	}
	if store.tokenByPlaintext(t, otherSessionToken).IsRevoked() {
		t.Error("the user's other session was revoked, want logout to end only one session")
	}
	if len(store.tokensForUser(userID)) != 2 {
		t.Errorf("rows for the user = %d, want logout to stamp rather than delete",
			len(store.tokensForUser(userID)))
	}
}

func TestRevokeRefreshTokenSurfacesAFailedLookup(t *testing.T) {
	cause := errors.New("pq: connection refused")

	testCases := []struct {
		name         string
		arrangeStore func(t *testing.T, store *inMemoryTokenStore) string
		assertError  func(t *testing.T, err error)
	}{
		{
			name: "unknown token",
			arrangeStore: func(t *testing.T, store *inMemoryTokenStore) string {
				return "never-issued-refresh-token"
			},
			assertError: func(t *testing.T, err error) {
				notFound := ErrRefreshTokenNotFound{}
				if !errors.As(err, &notFound) {
					t.Fatalf("error = %v, want ErrRefreshTokenNotFound", err)
				}
			},
		},
		{
			name: "lookup failed",
			arrangeStore: func(t *testing.T, store *inMemoryTokenStore) string {
				store.lookupFailure = ErrRefreshTokenLookupFailed{Cause: cause}
				return store.seedToken(t, seedTokenParams{UserID: uuid.New()})
			},
			assertError: func(t *testing.T, err error) {
				lookupFailed := ErrRefreshTokenLookupFailed{}
				if !errors.As(err, &lookupFailed) {
					t.Fatalf("error = %v, want ErrRefreshTokenLookupFailed", err)
				}
				if !errors.Is(err, cause) {
					t.Error("the lookup failure does not unwrap to the driver error")
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			store := newInMemoryTokenStore()
			presentedToken := testCase.arrangeStore(t, store)

			err := revokeForTest(t, store, presentedToken)

			testCase.assertError(t, err)
			for _, storedToken := range store.rows {
				if storedToken.IsRevoked() {
					t.Error("a stored token was revoked, want a failed lookup to revoke nothing")
				}
			}
		})
	}
}

func TestRevokeRefreshTokenSurfacesARevokeFailureUnchanged(t *testing.T) {
	cause := errors.New("pq: connection refused")
	store := newInMemoryTokenStore()
	presentedToken := store.seedToken(t, seedTokenParams{UserID: uuid.New()})
	store.revokeFailure = ErrRevokeRefreshTokenFailed{Cause: cause}

	err := revokeForTest(t, store, presentedToken)

	revokeFailed := ErrRevokeRefreshTokenFailed{}
	if !errors.As(err, &revokeFailed) {
		t.Fatalf("error = %v, want ErrRevokeRefreshTokenFailed", err)
	}
	if !errors.Is(err, cause) {
		t.Error("the revoke failure does not unwrap to the driver error")
	}
	if store.tokenByPlaintext(t, presentedToken).IsRevoked() {
		t.Error("the presented token was revoked, want it left live on failure")
	}
}
