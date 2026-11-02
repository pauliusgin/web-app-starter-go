package users

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	readmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/models"
)

func sendSignIn(t *testing.T, stores *userStores, body string) *http.Response {
	t.Helper()

	return sendUserRequest(t, sendUserRequestParams{
		handler: stores.signInHandler(),
		method:  http.MethodPost,
		path:    "/api/v1/users/signin",
		body:    body,
	})
}

func TestSignInUserAnswersTheReadModelAndAFreshTokenPair(t *testing.T) {
	stores := newUserStores(t)
	userID := stores.seedUser(t, seedUserParams{email: testEmail})

	response := sendSignIn(t, stores, `{"email":"  Accountant@Example.COM ","password":"correct-horse"}`)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	body := decodeBody(t, response)
	if body["id"] != userID.String() || body["email"] != testEmail {
		t.Errorf("id/email = %v/%v, want the seeded user", body["id"], body["email"])
	}
	assertSignInMethods(t, body, readmodels.SignInMethodEmailPassword)
	signedInAt, err := time.Parse(time.RFC3339, fmt.Sprint(body["signinAt"]))
	if err != nil || !signedInAt.After(seededAt) {
		t.Errorf("signinAt = %v, want a fresh stamp past the seeded %s", body["signinAt"], seededAt)
	}
	for _, key := range []string{"emailVerifiedAt", "firstName", "lastName"} {
		if value, present := body[key]; !present || value != nil {
			t.Errorf("body[%q] = %v (present %v), want null", key, value, present)
		}
	}
	if _, carriesUpdatedAt := body["updatedAt"]; carriesUpdatedAt {
		t.Error("response carries updatedAt, want only the read model's fields")
	}
	assertRefreshTokenCookieSet(t, response)

	accessToken, _ := body["accessToken"].(string)
	claims, err := stores.jwt.ParseAccessToken(accessToken)
	if err != nil || claims.UserID != userID {
		t.Errorf("access token subject = %s (%v), want %s", claims.UserID, err, userID)
	}
}

func TestSignInUserAnswersTheProfileNames(t *testing.T) {
	stores := newUserStores(t)
	userID := stores.seedUser(t, seedUserParams{email: testEmail})
	stores.profileNames[userID] = [2]string{"Ada", "Lovelace"}

	response := sendSignIn(t, stores, `{"email":"accountant@example.com","password":"correct-horse"}`)

	body := decodeBody(t, response)
	if body["firstName"] != "Ada" || body["lastName"] != "Lovelace" {
		t.Errorf("names = %v %v, want Ada Lovelace", body["firstName"], body["lastName"])
	}
}

func TestSignInUserRejectsWithoutTokens(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   handlers.ErrorCode
	}{
		{name: "malformed json", body: `{"email":`, wantStatus: http.StatusBadRequest, wantCode: handlers.ErrorCodeRequestBodyInvalid},
		{
			name:       "invalid email",
			body:       `{"email":"not-an-email","password":"correct-horse"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodeEmailInvalid,
		},
		{
			name:       "missing password",
			body:       `{"email":"accountant@example.com"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodePasswordMissing,
		},
		{
			name:       "wrong password",
			body:       `{"email":"accountant@example.com","password":"wrong-horse"}`,
			wantStatus: http.StatusUnauthorized,
			wantCode:   handlers.ErrorCodeInvalidCredentials,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			stores := newUserStores(t)
			stores.seedUser(t, seedUserParams{email: testEmail})

			response := sendSignIn(t, stores, testCase.body)

			assertErrorCode(t, response, testCase.wantStatus, testCase.wantCode)
			assertNoRefreshTokenCookie(t, response)
		})
	}
}

func TestSignInUserAnswersInternalErrorWhenTheLookupFails(t *testing.T) {
	silenceHandlerLogger(t)
	stores := newUserStores(t)
	stores.seedUser(t, seedUserParams{email: testEmail})
	stores.users.LookupFailure = usererrors.ErrFindUserFailed{Cause: errDatabaseDown}

	response := sendSignIn(t, stores, `{"email":"accountant@example.com","password":"correct-horse"}`)

	assertErrorCode(t, response, http.StatusInternalServerError, handlers.ErrorCodeInternal)
	assertNoRefreshTokenCookie(t, response)
}
