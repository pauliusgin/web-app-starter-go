package users

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	readmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/models"
)

type refusingAccessTokenIssuer struct{}

func (refusingAccessTokenIssuer) IssueAccessToken(jwt.IssueAccessTokenParams) (string, error) {
	return "", errors.New("signing key unavailable")
}

func (refusingAccessTokenIssuer) ParseAccessToken(string) (jwt.AccessTokenClaims, error) {
	return jwt.AccessTokenClaims{}, nil
}

func sendSignUp(t *testing.T, stores *userStores, body string) *http.Response {
	t.Helper()

	return sendUserRequest(t, sendUserRequestParams{
		handler: stores.signUpHandler(),
		method:  http.MethodPost,
		path:    "/api/v1/users/signup",
		body:    body,
	})
}

func TestSignUpUserStoresTheNormalizedEmailAndAnswersAFreshTokenPair(t *testing.T) {
	stores := newUserStores(t)

	response := sendSignUp(t, stores, `{"email":"  Accountant@Example.COM ","password":"correct-horse"}`)

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusCreated)
	}
	if len(stores.users.Users) != 1 {
		t.Fatalf("stored users = %d, want 1", len(stores.users.Users))
	}
	body := decodeBody(t, response)
	if body["email"] != testEmail {
		t.Errorf("email = %v, want the normalized %q", body["email"], testEmail)
	}
	if _, carriesRefreshToken := body["refreshToken"]; carriesRefreshToken {
		t.Error("response body carries refreshToken, want it only in the cookie")
	}
	for _, nullKey := range []string{"emailVerifiedAt", "firstName", "lastName"} {
		if value, present := body[nullKey]; !present || value != nil {
			t.Errorf("body[%q] = %v (present %t), want an explicit null for a fresh account", nullKey, value, present)
		}
	}
	assertSignInMethods(t, body, readmodels.SignInMethodEmailPassword)
	for _, stampKey := range []string{"createdAt", "signinAt"} {
		if stamp, _ := body[stampKey].(string); stamp == "" {
			t.Errorf("body[%q] = %v, want the time sign-up stored", stampKey, body[stampKey])
		}
	}
	assertRefreshTokenCookieSet(t, response)

	accessToken, _ := body["accessToken"].(string)
	claims, err := stores.jwt.ParseAccessToken(accessToken)
	if err != nil {
		t.Fatalf("parsing the issued access token: %v", err)
	}
	if stored := stores.storedUser(t, claims.UserID); stored.Email != testEmail || body["id"] != stored.ID.String() {
		t.Errorf("token subject %s / body id %v, want the stored user", claims.UserID, body["id"])
	}
	if stores.challenges.linksSentTo(testEmail) != 1 {
		t.Errorf("verification links sent = %d, want 1", stores.challenges.linksSentTo(testEmail))
	}
}

func TestSignUpUserRejectsABadRequestWithoutStoringAnyone(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantCode handlers.ErrorCode
	}{
		{name: "malformed json", body: `{"email":`, wantCode: handlers.ErrorCodeRequestBodyInvalid},
		{name: "invalid email", body: `{"email":"not-an-email","password":"correct-horse"}`, wantCode: handlers.ErrorCodeEmailInvalid},
		{name: "short password", body: `{"email":"accountant@example.com","password":"short"}`, wantCode: handlers.ErrorCodePasswordTooShort},
		{
			name:     "long password",
			body:     `{"email":"accountant@example.com","password":"` + strings.Repeat("a", primitive.MaxPasswordLength+1) + `"}`,
			wantCode: handlers.ErrorCodePasswordTooLong,
		},
		{
			name:     "password with whitespace",
			body:     `{"email":"accountant@example.com","password":"correct horse"}`,
			wantCode: handlers.ErrorCodePasswordContainsWhitespace,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			stores := newUserStores(t)

			response := sendSignUp(t, stores, testCase.body)

			assertErrorCode(t, response, http.StatusBadRequest, testCase.wantCode)
			if len(stores.users.Users) != 0 {
				t.Errorf("stored users = %d, want none", len(stores.users.Users))
			}
		})
	}
}

func TestSignUpUserAnswersConflictForATakenEmail(t *testing.T) {
	stores := newUserStores(t)
	stores.seedUser(t, seedUserParams{email: testEmail})

	response := sendSignUp(t, stores, `{"email":"accountant@example.com","password":"correct-horse"}`)

	assertErrorCode(t, response, http.StatusConflict, handlers.ErrorCodeEmailTaken)
}

func TestSignUpUserAnswersInternalErrorWhenNoAccessTokenCanBeIssued(t *testing.T) {
	silenceHandlerLogger(t)
	stores := newUserStores(t)
	stores.jwt = refusingAccessTokenIssuer{}

	response := sendSignUp(t, stores, `{"email":"accountant@example.com","password":"correct-horse"}`)

	assertErrorCode(t, response, http.StatusInternalServerError, handlers.ErrorCodeInternal)
}
