package refreshtoken

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func revokeForTest(t *testing.T, store *InMemoryRefreshTokenRepository, presentedToken string) error {
	t.Helper()

	return Revoke(context.Background(),
		RevokeDependencies{ReadRepository: store, WriteRepository: store},
		RevokeParams{PresentedToken: presentedToken})
}

func TestRevokeRefreshTokenRevokesALiveToken(t *testing.T) {
	store := newInMemoryTokenStore()
	userID := uuid.New()
	presentedToken := seedToken(t, store, seedTokenParams{UserID: userID})
	otherSessionToken := seedToken(t, store, seedTokenParams{UserID: userID})

	if err := revokeForTest(t, store, presentedToken); err != nil {
		t.Fatalf("RevokeRefreshToken: %v", err)
	}

	if !tokenByPlaintext(t, store, presentedToken).IsRevoked() {
		t.Error("the presented token is still live, want it revoked")
	}
	if tokenByPlaintext(t, store, otherSessionToken).IsRevoked() {
		t.Error("the user's other session was revoked, want logout to end only one session")
	}
	if len(tokensForUser(store, userID)) != 2 {
		t.Errorf("rows for the user = %d, want logout to stamp rather than delete",
			len(tokensForUser(store, userID)))
	}
}

func TestRevokeRefreshTokenSurfacesAFailedLookup(t *testing.T) {
	cause := errors.New("pq: connection refused")

	testCases := []struct {
		name         string
		arrangeStore func(t *testing.T, store *InMemoryRefreshTokenRepository) string
		assertError  func(t *testing.T, err error)
	}{
		{
			name: "unknown token",
			arrangeStore: func(t *testing.T, store *InMemoryRefreshTokenRepository) string {
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
			arrangeStore: func(t *testing.T, store *InMemoryRefreshTokenRepository) string {
				store.FindFailure = ErrRefreshTokenLookupFailed{Cause: cause}
				return seedToken(t, store, seedTokenParams{UserID: uuid.New()})
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
			for _, storedToken := range store.Tokens {
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
	presentedToken := seedToken(t, store, seedTokenParams{UserID: uuid.New()})
	store.RevokeFailure = ErrRevokeRefreshTokenFailed{Cause: cause}

	err := revokeForTest(t, store, presentedToken)

	revokeFailed := ErrRevokeRefreshTokenFailed{}
	if !errors.As(err, &revokeFailed) {
		t.Fatalf("error = %v, want ErrRevokeRefreshTokenFailed", err)
	}
	if !errors.Is(err, cause) {
		t.Error("the revoke failure does not unwrap to the driver error")
	}
	if tokenByPlaintext(t, store, presentedToken).IsRevoked() {
		t.Error("the presented token was revoked, want it left live on failure")
	}
}
