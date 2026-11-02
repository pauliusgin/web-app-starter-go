package users

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/password"
)

const addPasswordBodyForTest = `{"password":"added-ledger-password"}`

func sendAddPassword(t *testing.T, stores *userStores, userID uuid.UUID, body string) *http.Response {
	t.Helper()

	return sendUserRequest(t, sendUserRequestParams{
		handler:  stores.addPasswordHandler(),
		method:   http.MethodPost,
		path:     "/api/v1/users/add-password",
		body:     body,
		asUserID: userID,
	})
}

func TestAddPasswordStoresThePasswordOnAGoogleOnlyAccount(t *testing.T) {
	stores := newUserStores(t)
	userID := stores.seedGoogleUser(testGoogleSubject)

	response := sendAddPassword(t, stores, userID, addPasswordBodyForTest)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	matches, err := password.Verify(password.VerificationParams{
		Password: "added-ledger-password",
		Hash:     stores.users.Users[userID].PasswordHash,
	})
	if err != nil || !matches {
		t.Errorf("stored hash verifies the added password = %t (%v), want true", matches, err)
	}
}

func TestAddPasswordRejectsWithoutAddingAPassword(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   handlers.ErrorCode
	}{
		{name: "malformed json", body: `{"password":`, wantStatus: http.StatusBadRequest, wantCode: handlers.ErrorCodeRequestBodyInvalid},
		{
			name:       "password too short",
			body:       `{"password":"short"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodePasswordTooShort,
		},
		{
			name:       "password too long",
			body:       `{"password":"` + strings.Repeat("a", 65) + `"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodePasswordTooLong,
		},
		{
			name:       "password with whitespace",
			body:       `{"password":"added ledger password"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodePasswordContainsWhitespace,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			stores := newUserStores(t)
			userID := stores.seedGoogleUser(testGoogleSubject)

			response := sendAddPassword(t, stores, userID, testCase.body)

			assertErrorCode(t, response, testCase.wantStatus, testCase.wantCode)
			if stores.users.Users[userID].PasswordHash != "" {
				t.Error("stored hash set, want the google-only account left without a password")
			}
		})
	}
}

func TestAddPasswordAnswersPasswordAlreadySetForAPasswordAccount(t *testing.T) {
	stores := newUserStores(t)
	userID := stores.seedUser(t, seedUserParams{email: testEmail})
	seededHash := stores.users.Users[userID].PasswordHash

	response := sendAddPassword(t, stores, userID, addPasswordBodyForTest)

	assertErrorCode(t, response, http.StatusConflict, handlers.ErrorCodePasswordAlreadySet)
	if stores.users.Users[userID].PasswordHash != seededHash {
		t.Error("stored hash changed, want it untouched")
	}
}

func TestAddPasswordAnswersGoogleAccountNotConnectedForAnAccountWithNeither(t *testing.T) {
	stores := newUserStores(t)
	userID := stores.seedGoogleUser("")

	response := sendAddPassword(t, stores, userID, addPasswordBodyForTest)

	assertErrorCode(t, response, http.StatusConflict, handlers.ErrorCodeGoogleAccountNotConnected)
	if stores.users.Users[userID].PasswordHash != "" {
		t.Error("stored hash set, want the account left without a password")
	}
}

func TestAddPasswordAnswersInternalErrorWhenTheAuthenticatedUserVanished(t *testing.T) {
	silenceHandlerLogger(t)

	response := sendAddPassword(t, newUserStores(t), uuid.New(), addPasswordBodyForTest)

	assertErrorCode(t, response, http.StatusInternalServerError, handlers.ErrorCodeInternal)
}
