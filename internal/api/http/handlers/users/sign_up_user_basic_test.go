package users

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/tokens"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
	readmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

const signUpBodyForTest = `{"email":"accountant@example.com","password":"correct-horse"}`

const issuedRefreshTokenForTest = "PSaJ0Vd0kX3xw0kQeCr9nTgqcQ8bJ9YbFq0Q3fF8Zpo"

var errCollaboratorFailedForTest = errors.New("collaborator failed")

var signedUpUserForTest = models.User{
	ID:        uuid.MustParse("6f1c9f2e-4a1b-4c3d-9e8f-0a1b2c3d4e5f"),
	Email:     "accountant@example.com",
	CreatedAt: time.Date(2026, 3, 14, 9, 26, 53, 0, time.UTC),
	UpdatedAt: time.Date(2026, 3, 14, 9, 26, 53, 0, time.UTC),
}

type refusingAccessTokenIssuer struct{}

func (i refusingAccessTokenIssuer) IssueAccessToken(jwt.IssueAccessTokenParams) (string, error) {
	return "", errCollaboratorFailedForTest
}

func (i refusingAccessTokenIssuer) ParseAccessToken(string) (jwt.AccessTokenClaims, error) {
	return jwt.AccessTokenClaims{}, nil
}

func silenceHandlerLogger(t *testing.T) {
	t.Helper()

	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
}

func decodeBody(t *testing.T, response *http.Response) map[string]any {
	t.Helper()

	body := map[string]any{}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decoding the response body: %v", err)
	}

	return body
}

func newAccessTokenIssuerForTest(t *testing.T) jwt.Issuer {
	t.Helper()

	issuer, err := jwt.NewAccessTokenIssuer(jwt.NewAccessTokenIssuerParams{Secret: "signup-test-secret"})
	if err != nil {
		t.Fatalf("building the access token issuer: %v", err)
	}

	return issuer
}

func signUpAsSignedUpUser(context.Context, usecases.SignUpUserParams) (usecases.SignUpUserResult, error) {
	return usecases.SignUpUserResult{User: signedUpUserForTest}, nil
}

func getSignedUpUserReadModel(_ context.Context, id uuid.UUID) (readmodels.UserReadModel, error) {
	return readmodels.UserReadModel{
		ID:        id,
		Email:     signedUpUserForTest.Email,
		CreatedAt: signedUpUserForTest.CreatedAt,
	}, nil
}

func issueRefreshTokenForTest(context.Context, refreshtoken.IssueParams) (refreshtoken.IssueResult, error) {
	return refreshtoken.IssueResult{
		Plaintext: issuedRefreshTokenForTest,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, nil
}

func newWorkingSignUpDependencies(t *testing.T) SignUpUserDependencies {
	t.Helper()

	return SignUpUserDependencies{
		SignUpUser:           signUpAsSignedUpUser,
		GetUserReadModelByID: getSignedUpUserReadModel,
		IssueRefreshToken:    issueRefreshTokenForTest,
		Jwt:                  newAccessTokenIssuerForTest(t),
	}
}

type signUpRequestParams struct {
	Body         string
	Dependencies SignUpUserDependencies
}

func signUpWithDependencies(t *testing.T, params signUpRequestParams) *http.Response {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/users/signup", strings.NewReader(params.Body))
	request.Header.Set("Content-Type", "application/json")

	SignUpUser(params.Dependencies)(recorder, request)

	response := recorder.Result()
	t.Cleanup(func() { response.Body.Close() })

	return response
}

func assertSignUpRefused(t *testing.T, response *http.Response, wantCode handlers.ErrorCode) {
	t.Helper()

	if response.StatusCode < http.StatusBadRequest {
		t.Fatalf("status = %d, want an error status", response.StatusCode)
	}

	body := decodeBody(t, response)
	if code, _ := body["code"].(string); code != string(wantCode) {
		t.Errorf(`body["code"] = %q, want %q`, code, wantCode)
	}
	if message, _ := body["message"].(string); message == "" {
		t.Error(`body["message"] is empty, want a message`)
	}
	if body["accessToken"] != nil || body["refreshToken"] != nil {
		t.Errorf("response carries tokens %v, want none", body)
	}
	assertNoRefreshTokenCookie(t, response)
}

func TestSignUpUserAnswersTheReadModelAndAFreshTokenPair(t *testing.T) {
	var refreshTokenOwners []uuid.UUID
	dependencies := newWorkingSignUpDependencies(t)
	dependencies.IssueRefreshToken = func(
		ctx context.Context,
		params refreshtoken.IssueParams,
	) (refreshtoken.IssueResult, error) {
		refreshTokenOwners = append(refreshTokenOwners, params.UserID)
		return issueRefreshTokenForTest(ctx, params)
	}

	response := signUpWithDependencies(t, signUpRequestParams{
		Body:         signUpBodyForTest,
		Dependencies: dependencies,
	})

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusCreated)
	}
	if len(refreshTokenOwners) != 1 || refreshTokenOwners[0] != signedUpUserForTest.ID {
		t.Errorf("refresh token owners = %v, want the one signed up user %s",
			refreshTokenOwners, signedUpUserForTest.ID)
	}

	body := decodeBody(t, response)
	if body["id"] != signedUpUserForTest.ID.String() {
		t.Errorf("id = %v, want %s", body["id"], signedUpUserForTest.ID)
	}
	if body["email"] != signedUpUserForTest.Email {
		t.Errorf("email = %v, want %q", body["email"], signedUpUserForTest.Email)
	}
	if body["createdAt"] != signedUpUserForTest.CreatedAt.Format(time.RFC3339) {
		t.Errorf("createdAt = %v, want %s", body["createdAt"], signedUpUserForTest.CreatedAt.Format(time.RFC3339))
	}
	for _, nullField := range []string{"emailVerifiedAt", "signinAt", "firstName", "lastName"} {
		value, present := body[nullField]
		if !present || value != nil {
			t.Errorf("%s = %v (present %t), want an explicit null", nullField, value, present)
		}
	}
	if _, carriesRefreshToken := body["refreshToken"]; carriesRefreshToken {
		t.Error("response body carries refreshToken, want it only in the cookie")
	}
	assertRefreshTokenCookieSet(t, response)

	accessToken, _ := body["accessToken"].(string)
	claims, err := dependencies.Jwt.ParseAccessToken(accessToken)
	if err != nil {
		t.Fatalf("parsing the issued access token: %v", err)
	}
	if claims.UserID != signedUpUserForTest.ID {
		t.Errorf("access token subject = %s, want %s", claims.UserID, signedUpUserForTest.ID)
	}
	if claims.Email != signedUpUserForTest.Email {
		t.Errorf("access token email = %q, want %q", claims.Email, signedUpUserForTest.Email)
	}
}

func TestSignUpUserRefusesWhenTheRequestCannotBeDecoded(t *testing.T) {
	signUpAttempted := false
	dependencies := newWorkingSignUpDependencies(t)
	dependencies.SignUpUser = func(ctx context.Context, params usecases.SignUpUserParams) (usecases.SignUpUserResult, error) {
		signUpAttempted = true
		return signUpAsSignedUpUser(ctx, params)
	}

	response := signUpWithDependencies(t, signUpRequestParams{
		Body:         `{"email":`,
		Dependencies: dependencies,
	})

	assertSignUpRefused(t, response, handlers.ErrorCodeRequestBodyInvalid)
	if signUpAttempted {
		t.Error("sign up attempted with an undecodable request")
	}
}

func TestSignUpUserRefusesWhenTheInputIsInvalid(t *testing.T) {
	signUpAttempted := false
	dependencies := newWorkingSignUpDependencies(t)
	dependencies.SignUpUser = func(ctx context.Context, params usecases.SignUpUserParams) (usecases.SignUpUserResult, error) {
		signUpAttempted = true
		return signUpAsSignedUpUser(ctx, params)
	}

	response := signUpWithDependencies(t, signUpRequestParams{
		Body:         `{"email":"not-an-email","password":"correct-horse"}`,
		Dependencies: dependencies,
	})

	assertSignUpRefused(t, response, handlers.ErrorCodeEmailInvalid)
	if signUpAttempted {
		t.Error("sign up attempted with input the validators rejected")
	}
}

func TestSignUpUserAnswersBadRequestForAPasswordOverTheMaximum(t *testing.T) {
	response := signUpWithDependencies(t, signUpRequestParams{
		Body: `{"email":"accountant@example.com","password":"` +
			strings.Repeat("a", primitive.MaxPasswordLength+1) + `"}`,
		Dependencies: newWorkingSignUpDependencies(t),
	})

	assertSignUpRefused(t, response, handlers.ErrorCodePasswordTooLong)
	if response.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", response.StatusCode, http.StatusBadRequest)
	}
}

func TestSignUpUserAnswersBadRequestForAPasswordWithWhitespace(t *testing.T) {
	response := signUpWithDependencies(t, signUpRequestParams{
		Body:         `{"email":"accountant@example.com","password":"correct horse"}`,
		Dependencies: newWorkingSignUpDependencies(t),
	})

	assertSignUpRefused(t, response, handlers.ErrorCodePasswordContainsWhitespace)
	if response.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", response.StatusCode, http.StatusBadRequest)
	}
}

func TestSignUpUserRefusesWhenSignUpFails(t *testing.T) {
	silenceHandlerLogger(t)

	dependencies := newWorkingSignUpDependencies(t)
	dependencies.SignUpUser = func(context.Context, usecases.SignUpUserParams) (usecases.SignUpUserResult, error) {
		return usecases.SignUpUserResult{}, errCollaboratorFailedForTest
	}

	response := signUpWithDependencies(t, signUpRequestParams{
		Body:         signUpBodyForTest,
		Dependencies: dependencies,
	})

	assertSignUpRefused(t, response, handlers.ErrorCodeInternal)
}

func TestSignUpUserRefusesWhenAccessTokenIssuingFails(t *testing.T) {
	silenceHandlerLogger(t)

	dependencies := newWorkingSignUpDependencies(t)
	dependencies.Jwt = refusingAccessTokenIssuer{}

	response := signUpWithDependencies(t, signUpRequestParams{
		Body:         signUpBodyForTest,
		Dependencies: dependencies,
	})

	assertSignUpRefused(t, response, handlers.ErrorCodeInternal)
}

func TestSignUpUserRefusesWhenRefreshTokenIssuingFails(t *testing.T) {
	silenceHandlerLogger(t)

	dependencies := newWorkingSignUpDependencies(t)
	dependencies.IssueRefreshToken = func(context.Context, refreshtoken.IssueParams) (refreshtoken.IssueResult, error) {
		return refreshtoken.IssueResult{}, errCollaboratorFailedForTest
	}

	response := signUpWithDependencies(t, signUpRequestParams{
		Body:         signUpBodyForTest,
		Dependencies: dependencies,
	})

	assertSignUpRefused(t, response, handlers.ErrorCodeInternal)
}

func TestSignUpUserRefusesWhenTheReadModelLookupFails(t *testing.T) {
	silenceHandlerLogger(t)

	dependencies := newWorkingSignUpDependencies(t)
	dependencies.GetUserReadModelByID = func(context.Context, uuid.UUID) (readmodels.UserReadModel, error) {
		return readmodels.UserReadModel{}, errCollaboratorFailedForTest
	}

	response := signUpWithDependencies(t, signUpRequestParams{
		Body:         signUpBodyForTest,
		Dependencies: dependencies,
	})

	assertSignUpRefused(t, response, handlers.ErrorCodeInternal)
}

func TestSignUpUserNormalizesTheEmailBeforeSignUp(t *testing.T) {
	cases := []struct {
		name      string
		rawEmail  string
		wantEmail string
	}{
		{name: "surrounding whitespace", rawEmail: "  bob@example.com ", wantEmail: "bob@example.com"},
		{name: "mixed case", rawEmail: "Bob@Example.COM", wantEmail: "bob@example.com"},
		{name: "whitespace and mixed case", rawEmail: "  Bob@Example.COM ", wantEmail: "bob@example.com"},
		{name: "already normalized", rawEmail: "bob@example.com", wantEmail: "bob@example.com"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			var signedUpEmails []string
			dependencies := newWorkingSignUpDependencies(t)
			dependencies.SignUpUser = func(ctx context.Context, params usecases.SignUpUserParams) (usecases.SignUpUserResult, error) {
				signedUpEmails = append(signedUpEmails, params.Email)
				return signUpAsSignedUpUser(ctx, params)
			}

			response := signUpWithDependencies(t, signUpRequestParams{
				Body:         `{"email":"` + testCase.rawEmail + `","password":"correct-horse"}`,
				Dependencies: dependencies,
			})

			if response.StatusCode != http.StatusCreated {
				t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusCreated)
			}
			if len(signedUpEmails) != 1 || signedUpEmails[0] != testCase.wantEmail {
				t.Errorf("signed up emails = %q, want [%q]", signedUpEmails, testCase.wantEmail)
			}
		})
	}
}

func assertRefreshTokenCookieSet(t *testing.T, response *http.Response) {
	t.Helper()

	for _, cookie := range response.Cookies() {
		if cookie.Name != tokens.RefreshTokenCookieName {
			continue
		}
		if cookie.Value != issuedRefreshTokenForTest {
			t.Errorf("refresh token cookie = %q, want %q", cookie.Value, issuedRefreshTokenForTest)
		}
		if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode {
			t.Errorf("refresh token cookie flags = %+v, want HttpOnly, Secure, SameSite=Strict", cookie)
		}
		if cookie.Path != tokens.RefreshTokenCookiePath {
			t.Errorf("refresh token cookie path = %q, want %q", cookie.Path, tokens.RefreshTokenCookiePath)
		}
		return
	}
	t.Error("response sets no refresh token cookie")
}

func assertNoRefreshTokenCookie(t *testing.T, response *http.Response) {
	t.Helper()

	for _, cookie := range response.Cookies() {
		if cookie.Name == tokens.RefreshTokenCookieName {
			t.Errorf("refresh token cookie = %q, want none on a rejected request", cookie.Value)
		}
	}
}
