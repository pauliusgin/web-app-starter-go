package tokens

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
)

var errCollaboratorFailedForTest = errors.New("collaborator failed")

type refusingAccessTokenIssuer struct{}

func (i refusingAccessTokenIssuer) IssueAccessToken(jwt.IssueAccessTokenParams) (string, error) {
	return "", errCollaboratorFailedForTest
}

func (i refusingAccessTokenIssuer) ParseAccessToken(string) (jwt.AccessTokenClaims, error) {
	return jwt.AccessTokenClaims{}, nil
}

func newAccessTokenIssuerForTest(t *testing.T) jwt.Issuer {
	t.Helper()

	issuer, err := jwt.NewAccessTokenIssuer(jwt.NewAccessTokenIssuerParams{Secret: "rotate-test-secret"})
	if err != nil {
		t.Fatalf("building the access token issuer: %v", err)
	}

	return issuer
}

func findTokenHolderByID(_ context.Context, id uuid.UUID) (models.User, error) {
	if id != tokenHolderForTest.ID {
		return models.User{}, usererrors.ErrUserNotFound{}
	}

	return tokenHolderForTest, nil
}

func newWorkingRotateDependencies(t *testing.T, store *refreshtoken.InMemoryRefreshTokenRepository) RotateRefreshTokenDependencies {
	t.Helper()

	return RotateRefreshTokenDependencies{
		RotateRefreshToken: rotateOver(store),
		FindUserByID:       findTokenHolderByID,
		Jwt:                newAccessTokenIssuerForTest(t),
	}
}

type rotateRequestParams struct {
	RefreshTokenCookie string
	Dependencies       RotateRefreshTokenDependencies
}

func rotateWithDependencies(t *testing.T, params rotateRequestParams) *httptest.ResponseRecorder {
	t.Helper()

	handler := RotateRefreshToken(params.Dependencies)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tokens/refresh", nil)
	if params.RefreshTokenCookie != "" {
		request.AddCookie(&http.Cookie{Name: RefreshTokenCookieName, Value: params.RefreshTokenCookie})
	}

	handler.ServeHTTP(recorder, request)

	return recorder
}

func decodeRotateResponse(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	body := map[string]any{}
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decoding the response body: %v", err)
	}

	return body
}

func assertRotationRefused(t *testing.T, recorder *httptest.ResponseRecorder, wantCode handlers.ErrorCode) {
	t.Helper()

	if recorder.Code < http.StatusBadRequest {
		t.Fatalf("status = %d, want an error status", recorder.Code)
	}

	body := map[string]any{}
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decoding the response body: %v", err)
	}
	if body["code"] != string(wantCode) {
		t.Errorf("code = %v, want %q", body["code"], wantCode)
	}
	if body["accessToken"] != nil || body["refreshToken"] != nil {
		t.Errorf("response carries tokens %v, want none", body)
	}
}

func TestRotateRefreshTokenReturnsAFreshTokenPair(t *testing.T) {
	store := newRefreshTokenStore()
	dependencies := newWorkingRotateDependencies(t, store)

	recorder := rotateWithDependencies(t, rotateRequestParams{
		RefreshTokenCookie: presentedRefreshTokenForTest,
		Dependencies:       dependencies,
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	assertPresentedTokenRevoked(t, store, true)

	cookie := findRefreshTokenCookie(t, recorder)
	replacement, stored := store.Tokens[token.Hash(cookie.Value)]
	if !stored || replacement.IsRevoked() || replacement.UserID != tokenHolderForTest.ID || cookie.MaxAge <= 0 {
		t.Errorf("refresh token cookie = %+v, want a live replacement stored for %s", cookie, tokenHolderForTest.ID)
	}

	body := decodeRotateResponse(t, recorder)
	if _, carriesRefreshToken := body["refreshToken"]; carriesRefreshToken {
		t.Error("response body carries refreshToken, want it only in the cookie")
	}

	accessToken, _ := body["accessToken"].(string)
	claims, err := dependencies.Jwt.ParseAccessToken(accessToken)
	if err != nil {
		t.Fatalf("parsing the issued access token: %v", err)
	}
	if claims.UserID != tokenHolderForTest.ID {
		t.Errorf("access token subject = %s, want %s", claims.UserID, tokenHolderForTest.ID)
	}
	if claims.Email != tokenHolderForTest.Email {
		t.Errorf("access token email = %q, want %q", claims.Email, tokenHolderForTest.Email)
	}
}

func TestRotateRefreshTokenRefusesWhenTheCookieIsAbsent(t *testing.T) {
	silenceRefreshTokenLogger(t)
	store := newRefreshTokenStore()

	recorder := rotateWithDependencies(t, rotateRequestParams{Dependencies: newWorkingRotateDependencies(t, store)})

	assertRotationRefused(t, recorder, handlers.ErrorCodeRefreshTokenInvalid)
	assertRefreshTokenCookieCleared(t, recorder)
	assertPresentedTokenRevoked(t, store, false)
}

func TestRotateRefreshTokenRefusesWhenRotationFails(t *testing.T) {
	silenceRefreshTokenLogger(t)
	store := newRefreshTokenStore()
	store.FindFailure = errCollaboratorFailedForTest

	recorder := rotateWithDependencies(t, rotateRequestParams{
		RefreshTokenCookie: presentedRefreshTokenForTest,
		Dependencies:       newWorkingRotateDependencies(t, store),
	})

	assertRotationRefused(t, recorder, handlers.ErrorCodeInternal)
}

func TestRotateRefreshTokenRefusesWhenTheTokenHolderIsNotFound(t *testing.T) {
	silenceRefreshTokenLogger(t)

	dependencies := newWorkingRotateDependencies(t, newRefreshTokenStore())
	dependencies.FindUserByID = func(context.Context, uuid.UUID) (models.User, error) {
		return models.User{}, errCollaboratorFailedForTest
	}

	recorder := rotateWithDependencies(t, rotateRequestParams{
		RefreshTokenCookie: presentedRefreshTokenForTest,
		Dependencies:       dependencies,
	})

	assertRotationRefused(t, recorder, handlers.ErrorCodeInternal)
}

func TestRotateRefreshTokenRefusesWhenAccessTokenIssuingFails(t *testing.T) {
	silenceRefreshTokenLogger(t)

	dependencies := newWorkingRotateDependencies(t, newRefreshTokenStore())
	dependencies.Jwt = refusingAccessTokenIssuer{}

	recorder := rotateWithDependencies(t, rotateRequestParams{
		RefreshTokenCookie: presentedRefreshTokenForTest,
		Dependencies:       dependencies,
	})

	assertRotationRefused(t, recorder, handlers.ErrorCodeInternal)
}

func TestRotateRefreshTokenClearsTheCookieWhenTheTokenIsUnknown(t *testing.T) {
	silenceRefreshTokenLogger(t)

	recorder := rotateWithDependencies(t, rotateRequestParams{
		RefreshTokenCookie: unknownRefreshTokenForTest,
		Dependencies:       newWorkingRotateDependencies(t, newRefreshTokenStore()),
	})

	assertRotationRefused(t, recorder, handlers.ErrorCodeRefreshTokenInvalid)
	assertRefreshTokenCookieCleared(t, recorder)
}
