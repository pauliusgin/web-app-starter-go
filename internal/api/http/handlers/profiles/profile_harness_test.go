package profiles

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/usecases"
	usermodels "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

const profilesPath = "/api/v1/profiles"

var testUserID = uuid.MustParse("1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15")

var storedAt = time.Date(2026, time.October, 12, 9, 0, 0, 0, time.UTC)

var errDatabaseDown = errors.New("database unreachable")

func storedProfile() models.Profile {
	return models.Profile{
		ID:        testUserID,
		FirstName: "Ada",
		LastName:  "Lovelace",
		CreatedAt: storedAt,
		UpdatedAt: storedAt,
	}
}

func ignoringDispatcher() events.Dispatcher {
	ignoreEvent := func(context.Context, events.NewEvent) error { return nil }

	return events.Dispatcher{Recorder: ignoreEvent, Publisher: ignoreEvent}
}

func createProfileHandler(repository *repositories.InMemoryProfileRepository) http.Handler {
	dependencies := usecases.CreateProfileDependencies{
		ProfileRepository: repository,
		EventDispatcher:   ignoringDispatcher(),
	}

	return CreateProfile(CreateProfileDependencies{
		CreateProfile: func(ctx context.Context, params usecases.CreateProfileParams) (usecases.CreateProfileResult, error) {
			return usecases.CreateProfile(ctx, dependencies, params)
		},
	})
}

func updateProfileHandler(repository *repositories.InMemoryProfileRepository) http.Handler {
	dependencies := usecases.UpdateProfileDependencies{
		ProfileRepository: repository,
		EventDispatcher:   ignoringDispatcher(),
	}

	return UpdateProfile(UpdateProfileDependencies{
		UpdateProfile: func(ctx context.Context, params usecases.UpdateProfileParams) (usecases.UpdateProfileResult, error) {
			return usecases.UpdateProfile(ctx, dependencies, params)
		},
	})
}

func authenticateForTest(userID uuid.UUID) func(http.Handler) http.Handler {
	return middleware.RequireAuthenticatedUser(middleware.RequireAuthenticatedUserDependencies{
		ParseAccessToken: func(string) (jwt.AccessTokenClaims, error) {
			return jwt.AccessTokenClaims{UserID: userID}, nil
		},
		FindUserByID: func(_ context.Context, id uuid.UUID) (usermodels.User, error) {
			return usermodels.User{ID: id, Email: "accountant@example.com"}, nil
		},
	})
}

type sendProfileRequestParams struct {
	handler   http.Handler
	method    string
	body      string
	skipGuard bool
}

func sendProfileRequest(t *testing.T, params sendProfileRequestParams) *http.Response {
	t.Helper()

	served := params.handler
	if !params.skipGuard {
		served = authenticateForTest(testUserID)(params.handler)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(params.method, profilesPath, strings.NewReader(params.body))
	request.Header.Set("Authorization", "Bearer header.payload.signature")
	request.Header.Set("Content-Type", "application/json")

	served.ServeHTTP(recorder, request)

	response := recorder.Result()
	t.Cleanup(func() { response.Body.Close() })

	return response
}

func decodeBody(t *testing.T, response *http.Response) map[string]any {
	t.Helper()

	body := map[string]any{}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decoding the response body: %v", err)
	}

	return body
}

func assertErrorCode(t *testing.T, response *http.Response, wantStatus int, wantCode string) {
	t.Helper()

	if response.StatusCode != wantStatus {
		t.Fatalf("status = %d, want %d", response.StatusCode, wantStatus)
	}
	if code, _ := decodeBody(t, response)["code"].(string); code != wantCode {
		t.Errorf(`body["code"] = %q, want %q`, code, wantCode)
	}
}

func silenceHandlerLogger(t *testing.T) {
	t.Helper()

	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
}

var invalidProfileBodyCases = []struct {
	name     string
	body     string
	wantCode string
}{
	{name: "malformed json", body: `{"firstName":`, wantCode: "request_body_invalid"},
	{name: "unknown field", body: `{"firstName":"Ada","lastName":"Lovelace","age":36}`,
		wantCode: "request_body_invalid"},
	{name: "missing first name", body: `{"lastName":"Lovelace"}`, wantCode: "first_name_missing"},
	{name: "blank first name", body: `{"firstName":"  ","lastName":"Lovelace"}`, wantCode: "first_name_missing"},
	{name: "missing last name", body: `{"firstName":"Ada"}`, wantCode: "last_name_missing"},
	{name: "empty last name", body: `{"firstName":"Ada","lastName":""}`, wantCode: "last_name_missing"},
}
