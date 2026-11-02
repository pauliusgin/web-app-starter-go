package users

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/password"
)

const changePasswordBodyForTest = `{"currentPassword":"correct-horse","newPassword":"new-ledger-password"}`

func sendChangePassword(t *testing.T, stores *userStores, userID uuid.UUID, body string) *http.Response {
	t.Helper()

	return sendUserRequest(t, sendUserRequestParams{
		handler:  stores.changePasswordHandler(),
		method:   http.MethodPost,
		path:     "/api/v1/users/change-password",
		body:     body,
		asUserID: userID,
	})
}

func TestChangePasswordStoresTheNewPassword(t *testing.T) {
	stores := newUserStores(t)
	userID := stores.seedUser(t, seedUserParams{email: testEmail})

	response := sendChangePassword(t, stores, userID, changePasswordBodyForTest)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	matches, err := password.Verify(password.VerificationParams{
		Password: "new-ledger-password",
		Hash:     stores.users.Users[userID].PasswordHash,
	})
	if err != nil || !matches {
		t.Errorf("stored hash verifies the new password = %t (%v), want true", matches, err)
	}
}

func TestChangePasswordRejectsWithoutChangingThePassword(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   handlers.ErrorCode
	}{
		{name: "malformed json", body: `{"currentPassword":`, wantStatus: http.StatusBadRequest, wantCode: handlers.ErrorCodeRequestBodyInvalid},
		{
			name:       "missing current password",
			body:       `{"newPassword":"new-ledger-password"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodePasswordMissing,
		},
		{
			name:       "new password too short",
			body:       `{"currentPassword":"correct-horse","newPassword":"short"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodePasswordTooShort,
		},
		{
			name:       "new password too long",
			body:       `{"currentPassword":"correct-horse","newPassword":"` + strings.Repeat("a", 65) + `"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodePasswordTooLong,
		},
		{
			name:       "new password with whitespace",
			body:       `{"currentPassword":"correct-horse","newPassword":"new ledger password"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodePasswordContainsWhitespace,
		},
		{
			name:       "current password incorrect",
			body:       `{"currentPassword":"wrong-horse","newPassword":"new-ledger-password"}`,
			wantStatus: http.StatusForbidden,
			wantCode:   handlers.ErrorCodeCurrentPasswordIncorrect,
		},
		{
			name:       "new password equal to the current one",
			body:       `{"currentPassword":"correct-horse","newPassword":"correct-horse"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodePasswordUnchanged,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			stores := newUserStores(t)
			userID := stores.seedUser(t, seedUserParams{email: testEmail})
			seededHash := stores.users.Users[userID].PasswordHash

			response := sendChangePassword(t, stores, userID, testCase.body)

			assertErrorCode(t, response, testCase.wantStatus, testCase.wantCode)
			if stores.users.Users[userID].PasswordHash != seededHash {
				t.Error("stored hash changed, want it untouched")
			}
		})
	}
}

func TestChangePasswordAnswersInternalErrorWhenTheAuthenticatedUserVanished(t *testing.T) {
	silenceHandlerLogger(t)

	response := sendChangePassword(t, newUserStores(t), uuid.New(), changePasswordBodyForTest)

	assertErrorCode(t, response, http.StatusInternalServerError, handlers.ErrorCodeInternal)
}

func TestChangePasswordAnswersPasswordNotSetForAGoogleOnlyAccount(t *testing.T) {
	stores := newUserStores(t)
	userID := stores.seedGoogleUser(testGoogleSubject)

	response := sendChangePassword(t, stores, userID, changePasswordBodyForTest)

	assertErrorCode(t, response, http.StatusConflict, handlers.ErrorCodePasswordNotSet)
	if stores.users.Users[userID].PasswordHash != "" {
		t.Error("stored hash set, want the google-only account left without a password")
	}
}
