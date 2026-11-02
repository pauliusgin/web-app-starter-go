package users

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
)

// Handlers behind RequireAuthenticatedUser must fail closed when wired onto a route without it.
func TestAuthenticatedUserHandlersAnswerInternalErrorWhenTheRouteIsUnprotected(t *testing.T) {
	cases := []struct {
		name    string
		handler func(stores *userStores) http.Handler
		body    string
	}{
		{name: "verify email", handler: (*userStores).verifyEmailHandler},
		{name: "get user", handler: (*userStores).getUserByIDHandler},
		{name: "delete user", handler: (*userStores).deleteUserHandler},
		{name: "request email change", handler: (*userStores).requestEmailChangeHandler, body: `{"newEmail":"bookkeeper@example.com"}`},
		{name: "change password", handler: (*userStores).changePasswordHandler, body: changePasswordBodyForTest},
		{name: "add password", handler: (*userStores).addPasswordHandler, body: addPasswordBodyForTest},
		{name: "add google account", handler: (*userStores).addGoogleAccountHandler, body: verifiedGoogleIDTokenBody},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			silenceHandlerLogger(t)
			stores := newUserStores(t)
			userID := stores.seedUser(t, seedUserParams{email: testEmail})
			seeded := stores.users.Users[userID]

			response := sendUserRequest(t, sendUserRequestParams{
				handler:  testCase.handler(stores),
				method:   http.MethodPost,
				path:     "/api/v1/users/unprotected?token=" + presentedVerificationToken,
				body:     testCase.body,
				asUserID: uuid.Nil,
			})

			assertErrorCode(t, response, http.StatusInternalServerError, handlers.ErrorCodeInternal)
			if stores.users.Users[userID] != seeded || len(stores.challenges.Challenges) != 0 {
				t.Error("the stores changed, want an unprotected request to touch nothing")
			}
		})
	}
}
