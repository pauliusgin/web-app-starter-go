package users

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
)

const presentedVerificationToken = "Kq7wL2mZ9xR4tV6yB1nH8pD3sG5jF0aC"

func sendVerifyEmail(t *testing.T, stores *userStores, userID uuid.UUID, query string) *http.Response {
	t.Helper()

	return sendUserRequest(t, sendUserRequestParams{
		handler:  stores.verifyEmailHandler(),
		method:   http.MethodPost,
		path:     "/api/v1/users/verify-email" + query,
		asUserID: userID,
	})
}

func TestVerifyEmailRedeemsTheLinkForTheAuthenticatedUser(t *testing.T) {
	stores := newUserStores(t)
	userID := stores.seedUser(t, seedUserParams{email: testEmail})
	stores.challenges.seedLink(userID, testEmail, presentedVerificationToken)

	response := sendVerifyEmail(t, stores, userID, "?token="+presentedVerificationToken)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if !stores.storedUser(t, userID).IsEmailVerified() {
		t.Error("IsEmailVerified() = false, want the link to verify the user")
	}
}

func TestVerifyEmailRejectsALinkItCannotRedeem(t *testing.T) {
	cases := []struct {
		name       string
		arrange    func(t *testing.T, stores *userStores, userID uuid.UUID) string
		wantStatus int
		wantCode   handlers.ErrorCode
	}{
		{
			name:       "no token",
			arrange:    func(*testing.T, *userStores, uuid.UUID) string { return "" },
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodeVerificationTokenMissing,
		},
		{
			name:       "token matching no link",
			arrange:    func(*testing.T, *userStores, uuid.UUID) string { return "?token=" + presentedVerificationToken },
			wantStatus: http.StatusBadRequest,
			wantCode:   handlers.ErrorCodeVerificationLinkInvalid,
		},
		{
			name: "another user's link",
			arrange: func(_ *testing.T, stores *userStores, _ uuid.UUID) string {
				stores.challenges.seedLink(uuid.New(), "stranger@example.com", presentedVerificationToken)
				return "?token=" + presentedVerificationToken
			},
			wantStatus: http.StatusForbidden,
			wantCode:   handlers.ErrorCodeVerificationLinkNotOwned,
		},
		{
			name: "new address claimed since the request",
			arrange: func(t *testing.T, stores *userStores, userID uuid.UUID) string {
				stores.challenges.seedLink(userID, "bookkeeper@example.com", presentedVerificationToken)
				stores.seedUser(t, seedUserParams{email: "bookkeeper@example.com"})
				return "?token=" + presentedVerificationToken
			},
			wantStatus: http.StatusConflict,
			wantCode:   handlers.ErrorCodeEmailTaken,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			silenceHandlerLogger(t)
			stores := newUserStores(t)
			userID := stores.seedUser(t, seedUserParams{email: testEmail})

			response := sendVerifyEmail(t, stores, userID, testCase.arrange(t, stores, userID))

			assertErrorCode(t, response, testCase.wantStatus, testCase.wantCode)
			if stores.storedUser(t, userID).IsEmailVerified() {
				t.Error("IsEmailVerified() = true, want a rejected link to change nothing")
			}
		})
	}
}

func TestVerifyEmailAnswersInternalErrorWhenTheLookupFails(t *testing.T) {
	silenceHandlerLogger(t)
	stores := newUserStores(t)
	userID := stores.seedUser(t, seedUserParams{email: testEmail})
	stores.users.LookupFailure = usererrors.ErrFindUserFailed{Cause: errDatabaseDown}

	response := sendVerifyEmail(t, stores, userID, "?token="+presentedVerificationToken)

	assertErrorCode(t, response, http.StatusInternalServerError, handlers.ErrorCodeInternal)
}
