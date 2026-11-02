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

func sendSignInGoogle(t *testing.T, stores *userStores, body string) *http.Response {
	t.Helper()

	return sendUserRequest(t, sendUserRequestParams{
		handler: stores.signInGoogleHandler(),
		method:  http.MethodPost,
		path:    "/api/v1/users/google-signin",
		body:    body,
	})
}

func TestSignInUserGoogleAnswersTheReadModelAndAFreshTokenPair(t *testing.T) {
	stores := newUserStores(t)
	userID := stores.seedGoogleUser(testGoogleSubject)
	stores.profileNames[userID] = [2]string{"Ada", "Lovelace"}

	response := sendSignInGoogle(t, stores, verifiedGoogleIDTokenBody)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	body := decodeBody(t, response)
	if body["id"] != userID.String() || body["email"] != testEmail {
		t.Errorf("id/email = %v/%v, want the seeded google user", body["id"], body["email"])
	}
	if body["firstName"] != "Ada" || body["lastName"] != "Lovelace" {
		t.Errorf("names = %v %v, want Ada Lovelace", body["firstName"], body["lastName"])
	}
	assertSignInMethods(t, body, readmodels.SignInMethodGoogle)
	signedInAt, err := time.Parse(time.RFC3339, fmt.Sprint(body["signinAt"]))
	if err != nil || !signedInAt.After(seededAt) {
		t.Errorf("signinAt = %v, want a fresh stamp past the seeded %s", body["signinAt"], seededAt)
	}
	assertRefreshTokenCookieSet(t, response)

	accessToken, _ := body["accessToken"].(string)
	claims, err := stores.jwt.ParseAccessToken(accessToken)
	if err != nil || claims.UserID != userID {
		t.Errorf("access token subject = %s (%v), want %s", claims.UserID, err, userID)
	}
}

func TestSignInUserGoogleAnswersNullProfileNamesWithoutAProfile(t *testing.T) {
	stores := newUserStores(t)
	stores.seedGoogleUser(testGoogleSubject)

	response := sendSignInGoogle(t, stores, verifiedGoogleIDTokenBody)

	body := decodeBody(t, response)
	for _, key := range []string{"firstName", "lastName"} {
		if value, present := body[key]; !present || value != nil {
			t.Errorf("body[%q] = %v (present %v), want null", key, value, present)
		}
	}
}

func TestSignInUserGoogleRejectsWithoutTokens(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		seed       func(stores *userStores)
		wantStatus int
		wantCode   handlers.ErrorCode
	}{
		{
			name:       "malformed json",
			body:       `{"idToken":`,
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodeRequestBodyInvalid,
		},
		{
			name:       "missing ID token",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodeGoogleIDTokenMissing,
		},
		{
			name:       "invalid ID token",
			body:       `{"idToken":"forged-token"}`,
			wantStatus: http.StatusUnauthorized,
			wantCode:   handlers.ErrorCodeGoogleIDTokenInvalid,
		},
		{
			name:       "unverified google email",
			body:       `{"idToken":"` + unverifiedGoogleIDToken + `"}`,
			wantStatus: http.StatusForbidden,
			wantCode:   handlers.ErrorCodeGoogleEmailUnverified,
		},
		{
			name:       "google account not signed up",
			body:       verifiedGoogleIDTokenBody,
			wantStatus: http.StatusNotFound,
			wantCode:   handlers.ErrorCodeGoogleAccountNotFound,
		},
		{
			name:       "email held by a password account",
			body:       verifiedGoogleIDTokenBody,
			seed:       func(stores *userStores) { stores.seedUser(t, seedUserParams{email: testEmail}) },
			wantStatus: http.StatusConflict,
			wantCode:   handlers.ErrorCodeGoogleAccountNotLinked,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			stores := newUserStores(t)
			if testCase.seed != nil {
				testCase.seed(stores)
			}

			response := sendSignInGoogle(t, stores, testCase.body)

			assertErrorCode(t, response, testCase.wantStatus, testCase.wantCode)
			assertNoRefreshTokenCookie(t, response)
		})
	}
}

func TestSignInUserGoogleAnswersInternalErrorWhenTheLookupFails(t *testing.T) {
	silenceHandlerLogger(t)
	stores := newUserStores(t)
	stores.seedGoogleUser(testGoogleSubject)
	stores.users.LookupFailure = usererrors.ErrFindUserFailed{Cause: errDatabaseDown}

	response := sendSignInGoogle(t, stores, verifiedGoogleIDTokenBody)

	assertErrorCode(t, response, http.StatusInternalServerError, handlers.ErrorCodeInternal)
	assertNoRefreshTokenCookie(t, response)
}
