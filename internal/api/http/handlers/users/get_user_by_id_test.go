package users

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	readmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/models"
)

func sendGetUserByID(t *testing.T, stores *userStores, userID uuid.UUID) *http.Response {
	t.Helper()

	return sendUserRequest(t, sendUserRequestParams{
		handler:  stores.getUserByIDHandler(),
		method:   http.MethodGet,
		path:     "/api/v1/users/me",
		asUserID: userID,
	})
}

func TestGetUserByIDAnswersExactlyTheAuthenticatedUsersReadModel(t *testing.T) {
	stores := newUserStores(t)
	userID := stores.seedUser(t, seedUserParams{email: testEmail, verified: true})
	stores.profileNames[userID] = [2]string{"Ada", "Lovelace"}
	storedWithGoogle := stores.users.Users[userID]
	storedWithGoogle.GoogleSubject = testGoogleSubject
	stores.users.Users[userID] = storedWithGoogle

	response := sendGetUserByID(t, stores, userID)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	body := decodeBody(t, response)
	want := map[string]any{
		"id":              userID.String(),
		"email":           testEmail,
		"createdAt":       "2026-03-14T09:26:53Z",
		"emailVerifiedAt": "2026-03-14T09:26:53Z",
		"signinAt":        "2026-03-14T09:26:53Z",
		"firstName":       "Ada",
		"lastName":        "Lovelace",
	}
	assertSignInMethods(t, body, readmodels.SignInMethodEmailPassword, readmodels.SignInMethodGoogle)
	delete(body, "signInMethods")
	if len(body) != len(want) {
		t.Errorf("body = %v, want exactly the keys of %v", body, want)
	}
	for key, wantValue := range want {
		if body[key] != wantValue {
			t.Errorf("%s = %v, want %v", key, body[key], wantValue)
		}
	}
}

func TestGetUserByIDSendsNullForValuesTheUserDoesNotHaveYet(t *testing.T) {
	stores := newUserStores(t)
	userID := stores.seedUser(t, seedUserParams{email: testEmail})

	body := decodeBody(t, sendGetUserByID(t, stores, userID))

	for _, key := range []string{"emailVerifiedAt", "firstName", "lastName"} {
		if value, present := body[key]; !present || value != nil {
			t.Errorf("body[%q] = %v (present %v), want null", key, value, present)
		}
	}
}

func TestGetUserByIDRejectsAnAccountItCannotRead(t *testing.T) {
	t.Run("account gone", func(t *testing.T) {
		silenceHandlerLogger(t)

		response := sendGetUserByID(t, newUserStores(t), uuid.New())

		assertErrorCode(t, response, http.StatusNotFound, handlers.ErrorCodeUserNotFound)
	})

	t.Run("lookup fails", func(t *testing.T) {
		silenceHandlerLogger(t)
		stores := newUserStores(t)
		userID := stores.seedUser(t, seedUserParams{email: testEmail})
		stores.users.LookupFailure = usererrors.ErrFindUserFailed{Cause: errDatabaseDown}

		response := sendGetUserByID(t, stores, userID)

		assertErrorCode(t, response, http.StatusInternalServerError, handlers.ErrorCodeInternal)
	})
}
