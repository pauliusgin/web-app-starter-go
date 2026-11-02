package users

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
)

func sendDeleteUser(t *testing.T, stores *userStores, userID uuid.UUID) *http.Response {
	t.Helper()

	return sendUserRequest(t, sendUserRequestParams{
		handler:  stores.deleteUserHandler(),
		method:   http.MethodDelete,
		path:     "/api/v1/users/me",
		asUserID: userID,
	})
}

func TestDeleteUserDeletesTheAuthenticatedUsersAccount(t *testing.T) {
	stores := newUserStores(t)
	userID := stores.seedUser(t, seedUserParams{email: testEmail})

	response := sendDeleteUser(t, stores, userID)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if _, deleted := stores.users.SoftDeleted[userID]; !deleted {
		t.Error("the authenticated user was not soft-deleted")
	}
}

func TestDeleteUserRejectsAnAccountItCannotDelete(t *testing.T) {
	t.Run("account gone", func(t *testing.T) {
		silenceHandlerLogger(t)

		response := sendDeleteUser(t, newUserStores(t), uuid.New())

		assertErrorCode(t, response, http.StatusNotFound, handlers.ErrorCodeUserNotFound)
	})

	t.Run("deletion fails", func(t *testing.T) {
		silenceHandlerLogger(t)
		stores := newUserStores(t)
		userID := stores.seedUser(t, seedUserParams{email: testEmail})
		stores.users.SoftDeleteFailure = usererrors.ErrSoftDeleteUserFailed{Cause: errDatabaseDown}

		response := sendDeleteUser(t, stores, userID)

		assertErrorCode(t, response, http.StatusInternalServerError, handlers.ErrorCodeInternal)
		stores.storedUser(t, userID)
	})
}
