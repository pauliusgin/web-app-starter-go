package users

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	readmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/queries"
)

var currentUserIDForTest = uuid.MustParse("2b8f4d6a-1c3e-4a5b-9d7f-6e0a1b2c3d4e")

var currentUserCreatedAtForTest = time.Date(2026, time.March, 14, 9, 26, 53, 0, time.UTC)

func getCurrentUserReadModel(_ context.Context, params queries.GetUserByIDParams) (queries.GetUserByIDResult, error) {
	return queries.GetUserByIDResult{User: readmodels.UserReadModel{
		ID:        params.UserID,
		Email:     "accountant@example.com",
		CreatedAt: currentUserCreatedAtForTest,
	}}, nil
}

type sendGetUserByIDParams struct {
	getUserByID GetUserByIDUseCase
	skipGuard   bool
}

func sendGetUserByID(t *testing.T, params sendGetUserByIDParams) *http.Response {
	t.Helper()

	var served http.Handler = GetUserByID(GetUserByIDDependencies{GetUserByID: params.getUserByID})
	if !params.skipGuard {
		served = authenticateForTest(currentUserIDForTest)(served)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	request.Header.Set("Authorization", "Bearer header.payload.signature")

	served.ServeHTTP(recorder, request)

	response := recorder.Result()
	t.Cleanup(func() { response.Body.Close() })

	return response
}

func TestGetUserByIDAnswersTheAuthenticatedUsersReadModel(t *testing.T) {
	response := sendGetUserByID(t, sendGetUserByIDParams{getUserByID: getCurrentUserReadModel})

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	body := decodeBody(t, response)
	if body["id"] != currentUserIDForTest.String() {
		t.Errorf("id = %v, want %s", body["id"], currentUserIDForTest)
	}
	if body["email"] != "accountant@example.com" {
		t.Errorf("email = %v, want accountant@example.com", body["email"])
	}
	if body["createdAt"] != currentUserCreatedAtForTest.Format(time.RFC3339) {
		t.Errorf("createdAt = %v, want %s", body["createdAt"], currentUserCreatedAtForTest.Format(time.RFC3339))
	}
	for _, nullField := range []string{"emailVerifiedAt", "signinAt", "firstName", "lastName"} {
		value, present := body[nullField]
		if !present || value != nil {
			t.Errorf("%s = %v (present %t), want an explicit null", nullField, value, present)
		}
	}
}

func TestGetUserByIDRejectsAnAccountItCannotRead(t *testing.T) {
	cases := []struct {
		name       string
		failure    error
		wantStatus int
		wantCode   handlers.ErrorCode
	}{
		{
			name:       "account gone",
			failure:    usererrors.ErrUserNotFound{ID: currentUserIDForTest},
			wantStatus: http.StatusNotFound,
			wantCode:   handlers.ErrorCodeUserNotFound,
		},
		{
			name:       "lookup fails",
			failure:    usererrors.ErrFindUserFailed{ID: currentUserIDForTest, Cause: errCollaboratorFailedForTest},
			wantStatus: http.StatusInternalServerError,
			wantCode:   handlers.ErrorCodeInternal,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			silenceHandlerLogger(t)

			response := sendGetUserByID(t, sendGetUserByIDParams{
				getUserByID: func(context.Context, queries.GetUserByIDParams) (queries.GetUserByIDResult, error) {
					return queries.GetUserByIDResult{}, testCase.failure
				},
			})

			if response.StatusCode != testCase.wantStatus {
				t.Fatalf("status = %d, want %d", response.StatusCode, testCase.wantStatus)
			}
			if code := decodeBody(t, response)["code"]; code != string(testCase.wantCode) {
				t.Errorf("code = %v, want %q", code, testCase.wantCode)
			}
		})
	}
}

func TestGetUserByIDAnswersInternalErrorWhenTheRouteIsUnprotected(t *testing.T) {
	silenceHandlerLogger(t)

	response := sendGetUserByID(t, sendGetUserByIDParams{getUserByID: getCurrentUserReadModel, skipGuard: true})

	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusInternalServerError)
	}
}
