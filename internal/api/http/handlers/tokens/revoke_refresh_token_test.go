package tokens

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
)

type revokeRequestParams struct {
	RefreshTokenCookie string
	Store              *refreshtoken.InMemoryRefreshTokenRepository
}

func sendRevokeRequest(t *testing.T, params revokeRequestParams) *httptest.ResponseRecorder {
	t.Helper()

	handler := RevokeRefreshToken(RevokeRefreshTokenDependencies{RevokeRefreshToken: revokeOver(params.Store)})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tokens/revoke", nil)
	if params.RefreshTokenCookie != "" {
		request.AddCookie(&http.Cookie{Name: RefreshTokenCookieName, Value: params.RefreshTokenCookie})
	}

	handler.ServeHTTP(recorder, request)

	return recorder
}

func TestRevokeRefreshTokenAnswersOKWhenTheLiveTokenIsRevoked(t *testing.T) {
	store := newRefreshTokenStore()

	recorder := sendRevokeRequest(t, revokeRequestParams{RefreshTokenCookie: presentedRefreshTokenForTest, Store: store})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	assertPresentedTokenRevoked(t, store, true)
	assertRefreshTokenCookieCleared(t, recorder)
}

func TestRevokeRefreshTokenAnswersOKWhenTheTokenIsAlreadyGone(t *testing.T) {
	store := newRefreshTokenStore()

	recorder := sendRevokeRequest(t, revokeRequestParams{RefreshTokenCookie: unknownRefreshTokenForTest, Store: store})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d on a repeated logout", recorder.Code, http.StatusOK)
	}
	if challenge := recorder.Header().Get("WWW-Authenticate"); challenge != "" {
		t.Errorf("WWW-Authenticate = %q, want no challenge", challenge)
	}
	assertPresentedTokenRevoked(t, store, false)
	assertRefreshTokenCookieCleared(t, recorder)
}

func TestRevokeRefreshTokenAnswersOKWithoutACookie(t *testing.T) {
	store := newRefreshTokenStore()

	recorder := sendRevokeRequest(t, revokeRequestParams{Store: store})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	assertPresentedTokenRevoked(t, store, false)
	assertRefreshTokenCookieCleared(t, recorder)
}

func TestRevokeRefreshTokenAnswersServerErrorWhenTheDatabaseFails(t *testing.T) {
	silenceRefreshTokenLogger(t)
	store := newRefreshTokenStore()
	store.RevokeFailure = refreshtoken.ErrRevokeRefreshTokenFailed{Cause: context.DeadlineExceeded}

	recorder := sendRevokeRequest(t, revokeRequestParams{RefreshTokenCookie: presentedRefreshTokenForTest, Store: store})

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}
