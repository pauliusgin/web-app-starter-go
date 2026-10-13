package tokens

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
)

const replacementRefreshTokenForTest = "pR4nV8sT2wY6zA0cE3gJ5lM7oQ9uX1bD4fH6kN8qS0v"

var errCollaboratorFailedForTest = errors.New("collaborator failed")

var tokenHolderForTest = models.User{
	ID:    uuid.MustParse("5b0c7d4e-2f8a-4c61-9e3b-7a1d6f0e8c29"),
	Email: "holder@example.com",
}

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

func rotateToReplacementToken(context.Context, refreshtoken.RotateParams) (refreshtoken.Rotated, error) {
	return refreshtoken.Rotated{
		UserID:    tokenHolderForTest.ID,
		Plaintext: replacementRefreshTokenForTest,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil
}

func findTokenHolderByID(_ context.Context, id uuid.UUID) (models.User, error) {
	if id != tokenHolderForTest.ID {
		return models.User{}, usererrors.ErrUserNotFound{}
	}

	return tokenHolderForTest, nil
}

func newWorkingRotateDependencies(t *testing.T) RotateDependencies {
	t.Helper()

	return RotateDependencies{
		RotateRefreshToken: rotateToReplacementToken,
		FindUserByID:       findTokenHolderByID,
		Jwt:                newAccessTokenIssuerForTest(t),
	}
}

type rotateRequestParams struct {
	AuthorizationHeader string
	Dependencies        RotateDependencies
}

func rotateWithDependencies(t *testing.T, params rotateRequestParams) *httptest.ResponseRecorder {
	t.Helper()

	handler := RotateRefreshToken(params.Dependencies)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tokens/refresh", nil)
	if params.AuthorizationHeader != "" {
		request.Header.Set("Authorization", params.AuthorizationHeader)
	}

	handler.ServeHTTP(recorder, request)

	return recorder
}

func decodeRotateResponse(t *testing.T, recorder *httptest.ResponseRecorder) rotateRefreshTokenResponse {
	t.Helper()

	var body rotateRefreshTokenResponse
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
	var rotatedTokens []string
	dependencies := newWorkingRotateDependencies(t)
	dependencies.RotateRefreshToken = func(
		ctx context.Context,
		params refreshtoken.RotateParams,
	) (refreshtoken.Rotated, error) {
		rotatedTokens = append(rotatedTokens, params.PresentedToken)
		return rotateToReplacementToken(ctx, params)
	}

	recorder := rotateWithDependencies(t, rotateRequestParams{
		AuthorizationHeader: "Bearer " + presentedRefreshTokenForTest,
		Dependencies:        dependencies,
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if len(rotatedTokens) != 1 || rotatedTokens[0] != presentedRefreshTokenForTest {
		t.Errorf("rotations = %v, want the one presented token", rotatedTokens)
	}

	body := decodeRotateResponse(t, recorder)
	if body.RefreshToken != replacementRefreshTokenForTest {
		t.Errorf("refreshToken = %q, want the rotated replacement %q",
			body.RefreshToken, replacementRefreshTokenForTest)
	}

	claims, err := dependencies.Jwt.ParseAccessToken(body.AccessToken)
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

func TestRotateRefreshTokenRefusesWhenTheBearerTokenCannotBeRead(t *testing.T) {
	silenceRefreshTokenLogger(t)

	rotationAttempted := false
	dependencies := newWorkingRotateDependencies(t)
	dependencies.RotateRefreshToken = func(
		ctx context.Context,
		params refreshtoken.RotateParams,
	) (refreshtoken.Rotated, error) {
		rotationAttempted = true
		return rotateToReplacementToken(ctx, params)
	}

	recorder := rotateWithDependencies(t, rotateRequestParams{Dependencies: dependencies})

	assertRotationRefused(t, recorder, handlers.ErrorCodeRefreshTokenInvalid)
	if rotationAttempted {
		t.Error("rotation attempted without a readable bearer token")
	}
}

func TestRotateRefreshTokenRefusesWhenRotationFails(t *testing.T) {
	silenceRefreshTokenLogger(t)

	dependencies := newWorkingRotateDependencies(t)
	dependencies.RotateRefreshToken = func(context.Context, refreshtoken.RotateParams) (refreshtoken.Rotated, error) {
		return refreshtoken.Rotated{}, errCollaboratorFailedForTest
	}

	recorder := rotateWithDependencies(t, rotateRequestParams{
		AuthorizationHeader: "Bearer " + presentedRefreshTokenForTest,
		Dependencies:        dependencies,
	})

	assertRotationRefused(t, recorder, handlers.ErrorCodeInternal)
}

func TestRotateRefreshTokenRefusesWhenTheTokenHolderIsNotFound(t *testing.T) {
	silenceRefreshTokenLogger(t)

	dependencies := newWorkingRotateDependencies(t)
	dependencies.FindUserByID = func(context.Context, uuid.UUID) (models.User, error) {
		return models.User{}, errCollaboratorFailedForTest
	}

	recorder := rotateWithDependencies(t, rotateRequestParams{
		AuthorizationHeader: "Bearer " + presentedRefreshTokenForTest,
		Dependencies:        dependencies,
	})

	assertRotationRefused(t, recorder, handlers.ErrorCodeInternal)
}

func TestRotateRefreshTokenRefusesWhenAccessTokenIssuingFails(t *testing.T) {
	silenceRefreshTokenLogger(t)

	dependencies := newWorkingRotateDependencies(t)
	dependencies.Jwt = refusingAccessTokenIssuer{}

	recorder := rotateWithDependencies(t, rotateRequestParams{
		AuthorizationHeader: "Bearer " + presentedRefreshTokenForTest,
		Dependencies:        dependencies,
	})

	assertRotationRefused(t, recorder, handlers.ErrorCodeInternal)
}
