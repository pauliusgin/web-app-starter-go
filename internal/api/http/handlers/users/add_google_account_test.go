package users

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
)

func sendAddGoogleAccount(t *testing.T, stores *userStores, userID uuid.UUID, body string) *http.Response {
	t.Helper()

	return sendUserRequest(t, sendUserRequestParams{
		handler:  stores.addGoogleAccountHandler(),
		method:   http.MethodPost,
		path:     "/api/v1/users/add-google",
		body:     body,
		asUserID: userID,
	})
}

func TestAddGoogleAccountConnectsTheGoogleAccountToAPasswordAccount(t *testing.T) {
	stores := newUserStores(t)
	userID := stores.seedUser(t, seedUserParams{email: testEmail})

	response := sendAddGoogleAccount(t, stores, userID, verifiedGoogleIDTokenBody)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if stored := stores.users.Users[userID]; stored.GoogleSubject != testGoogleSubject {
		t.Errorf("GoogleSubject = %q, want %q", stored.GoogleSubject, testGoogleSubject)
	}
}

func TestAddGoogleAccountRejectsWithoutConnectingAGoogleAccount(t *testing.T) {
	cases := []struct {
		name       string
		email      string
		body       string
		wantStatus int
		wantCode   handlers.ErrorCode
	}{
		{name: "malformed json", email: testEmail, body: `{"idToken":`,
			wantStatus: http.StatusBadRequest, wantCode: handlers.ErrorCodeRequestBodyInvalid},
		{name: "missing id token", email: testEmail, body: `{"idToken":"  "}`,
			wantStatus: http.StatusBadRequest, wantCode: handlers.ErrorCodeGoogleIDTokenMissing},
		{name: "invalid id token", email: testEmail, body: `{"idToken":"forged-google-id-token"}`,
			wantStatus: http.StatusUnauthorized, wantCode: handlers.ErrorCodeGoogleIDTokenInvalid},
		{name: "unverified google email", email: testEmail, body: `{"idToken":"` + unverifiedGoogleIDToken + `"}`,
			wantStatus: http.StatusForbidden, wantCode: handlers.ErrorCodeGoogleEmailUnverified},
		{name: "google email differs", email: "bookkeeper@example.com", body: verifiedGoogleIDTokenBody,
			wantStatus: http.StatusConflict, wantCode: handlers.ErrorCodeGoogleEmailMismatch},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			stores := newUserStores(t)
			userID := stores.seedUser(t, seedUserParams{email: testCase.email})

			response := sendAddGoogleAccount(t, stores, userID, testCase.body)

			assertErrorCode(t, response, testCase.wantStatus, testCase.wantCode)
			if stores.users.Users[userID].GoogleSubject != "" {
				t.Error("google subject stored, want the password account left without one")
			}
		})
	}
}

func TestAddGoogleAccountAnswersGoogleAccountAlreadyConnectedForAGoogleAccount(t *testing.T) {
	stores := newUserStores(t)
	userID := stores.seedGoogleUser("earlier-google-subject")

	response := sendAddGoogleAccount(t, stores, userID, verifiedGoogleIDTokenBody)

	assertErrorCode(t, response, http.StatusConflict, handlers.ErrorCodeGoogleAccountAlreadyConnected)
	if stores.users.Users[userID].GoogleSubject != "earlier-google-subject" {
		t.Error("google subject changed, want the earlier one kept")
	}
}

func TestAddGoogleAccountAnswersGoogleAccountTakenWhenAnotherUserHoldsIt(t *testing.T) {
	stores := newUserStores(t)
	holderID := uuid.New()
	stores.users.Users[holderID] = repositories.InMemoryUser{
		User:          models.User{ID: holderID, Email: "bookkeeper@example.com", CreatedAt: seededAt},
		GoogleSubject: testGoogleSubject,
	}
	userID := stores.seedUser(t, seedUserParams{email: testEmail})

	response := sendAddGoogleAccount(t, stores, userID, verifiedGoogleIDTokenBody)

	assertErrorCode(t, response, http.StatusConflict, handlers.ErrorCodeGoogleAccountTaken)
	if stores.users.Users[userID].GoogleSubject != "" {
		t.Error("google subject stored, want the password account left without one")
	}
}

func TestAddGoogleAccountAnswersInternalErrorWhenTheAuthenticatedUserVanished(t *testing.T) {
	silenceHandlerLogger(t)

	response := sendAddGoogleAccount(t, newUserStores(t), uuid.New(), verifiedGoogleIDTokenBody)

	assertErrorCode(t, response, http.StatusInternalServerError, handlers.ErrorCodeInternal)
}
