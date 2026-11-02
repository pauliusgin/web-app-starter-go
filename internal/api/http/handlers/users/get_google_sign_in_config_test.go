package users

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/queries"
)

const googleClientIDForTest = "1234567890-abc123def456.apps.googleusercontent.com"

func sendGetGoogleSignInConfig(t *testing.T, handler http.Handler) *http.Response {
	t.Helper()

	return sendUserRequest(t, sendUserRequestParams{
		handler:  handler,
		method:   http.MethodGet,
		path:     "/api/v1/users/google-config",
		asUserID: uuid.Nil,
	})
}

func googleSignInConfigHandlerForTest() http.Handler {
	return GetGoogleSignInConfig(GetGoogleSignInConfigDependencies{
		GetGoogleSignInConfig: func(ctx context.Context) (queries.GetGoogleSignInConfigResult, error) {
			return queries.GetGoogleSignInConfig(ctx, queries.GetGoogleSignInConfigDependencies{
				GoogleClientID: googleClientIDForTest,
			})
		},
	})
}

func TestGetGoogleSignInConfigAnswersExactlyTheClientID(t *testing.T) {
	response := sendGetGoogleSignInConfig(t, googleSignInConfigHandlerForTest())

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	body := decodeBody(t, response)
	if len(body) != 1 || body["clientId"] != googleClientIDForTest {
		t.Errorf("body = %v, want exactly {clientId: %q}", body, googleClientIDForTest)
	}
}

func TestGetGoogleSignInConfigLetsBrowsersCacheTheAnswer(t *testing.T) {
	response := sendGetGoogleSignInConfig(t, googleSignInConfigHandlerForTest())

	if got := response.Header.Get("Cache-Control"); got != "public, max-age=3600" {
		t.Errorf("Cache-Control = %q, want %q", got, "public, max-age=3600")
	}
}

func TestGetGoogleSignInConfigAnswersInternalErrorWhenTheLookupFails(t *testing.T) {
	silenceHandlerLogger(t)
	failing := GetGoogleSignInConfig(GetGoogleSignInConfigDependencies{
		GetGoogleSignInConfig: func(context.Context) (queries.GetGoogleSignInConfigResult, error) {
			return queries.GetGoogleSignInConfigResult{}, errDatabaseDown
		},
	})

	response := sendGetGoogleSignInConfig(t, failing)

	assertErrorCode(t, response, http.StatusInternalServerError, handlers.ErrorCodeInternal)
	if got := response.Header.Get("Cache-Control"); got != "" {
		t.Errorf("Cache-Control = %q, want none on a failure", got)
	}
}
