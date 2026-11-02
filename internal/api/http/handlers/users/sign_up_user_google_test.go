package users

import (
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/googleidtoken"
	readmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/models"
)

func sendSignUpGoogle(t *testing.T, stores *userStores, body string) *http.Response {
	t.Helper()

	return sendUserRequest(t, sendUserRequestParams{
		handler: stores.signUpGoogleHandler(),
		method:  http.MethodPost,
		path:    "/api/v1/users/google-signup",
		body:    body,
	})
}

func TestSignUpUserGoogleAnswersTheVerifiedUserAndAFreshTokenPair(t *testing.T) {
	stores := newUserStores(t)

	response := sendSignUpGoogle(t, stores, verifiedGoogleIDTokenBody)

	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusCreated)
	}
	body := decodeBody(t, response)
	userID, err := uuid.Parse(body["id"].(string))
	if err != nil || body["email"] != testEmail {
		t.Fatalf("id/email = %v/%v, want a new user for %q", body["id"], body["email"], testEmail)
	}
	if stored := stores.storedUser(t, userID); !stored.IsEmailVerified() {
		t.Error("stored user is unverified, want google's verified email to count")
	}
	if body["emailVerifiedAt"] == nil || body["signinAt"] == nil {
		t.Errorf("emailVerifiedAt/signinAt = %v/%v, want both stamped", body["emailVerifiedAt"], body["signinAt"])
	}
	for _, key := range []string{"firstName", "lastName"} {
		if value, present := body[key]; !present || value != nil {
			t.Errorf("body[%q] = %v (present %v), want null", key, value, present)
		}
	}
	assertSignInMethods(t, body, readmodels.SignInMethodGoogle)
	assertRefreshTokenCookieSet(t, response)

	accessToken, _ := body["accessToken"].(string)
	claims, err := stores.jwt.ParseAccessToken(accessToken)
	if err != nil || claims.UserID != userID {
		t.Errorf("access token subject = %s (%v), want %s", claims.UserID, err, userID)
	}
}

func TestSignUpUserGoogleRejectsWithoutTokens(t *testing.T) {
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
			body:       `{"idToken":"  "}`,
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
			name:       "google account already signed up",
			body:       verifiedGoogleIDTokenBody,
			seed:       func(stores *userStores) { stores.seedGoogleUser(testGoogleSubject) },
			wantStatus: http.StatusConflict,
			wantCode:   handlers.ErrorCodeGoogleAccountTaken,
		},
		{
			name:       "email held by a password account",
			body:       verifiedGoogleIDTokenBody,
			seed:       func(stores *userStores) { stores.seedUser(t, seedUserParams{email: testEmail}) },
			wantStatus: http.StatusConflict,
			wantCode:   handlers.ErrorCodeEmailTaken,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			stores := newUserStores(t)
			if testCase.seed != nil {
				testCase.seed(stores)
			}
			usersBefore := len(stores.users.Users)

			response := sendSignUpGoogle(t, stores, testCase.body)

			assertErrorCode(t, response, testCase.wantStatus, testCase.wantCode)
			assertNoRefreshTokenCookie(t, response)
			if len(stores.users.Users) != usersBefore {
				t.Errorf("stored users = %d, want the %d there before", len(stores.users.Users), usersBefore)
			}
		})
	}
}

func TestSignUpUserGoogleAnswersInternalErrorWhenGoogleIsUnreachable(t *testing.T) {
	silenceHandlerLogger(t)
	stores := newUserStores(t)
	stores.google.VerifyFailure = googleidtoken.ErrIDTokenVerificationFailed{Cause: errors.New("dial tcp: timeout")}

	response := sendSignUpGoogle(t, stores, verifiedGoogleIDTokenBody)

	assertErrorCode(t, response, http.StatusInternalServerError, handlers.ErrorCodeInternal)
	assertNoRefreshTokenCookie(t, response)
}
