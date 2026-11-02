package users

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

var deletingUserID = uuid.MustParse("9b2e7d41-3c5a-4f80-a1d6-2e4c8b0f7a93")

type recordingUserDeleter struct {
	failure       error
	receivedCalls []usecases.DeleteUserParams
}

func (d *recordingUserDeleter) deleteUser(_ context.Context, params usecases.DeleteUserParams) error {
	d.receivedCalls = append(d.receivedCalls, params)

	return d.failure
}

type sendDeleteUserParams struct {
	deleter   *recordingUserDeleter
	skipGuard bool
}

func sendDeleteUser(t *testing.T, params sendDeleteUserParams) *httptest.ResponseRecorder {
	t.Helper()

	var served http.Handler = DeleteUser(DeleteUserDependencies{DeleteUser: params.deleter.deleteUser})
	if !params.skipGuard {
		served = authenticateForTest(deletingUserID)(served)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me", nil)
	request.Header.Set("Authorization", "Bearer header.payload.signature")

	served.ServeHTTP(recorder, request)

	return recorder
}

func TestDeleteUserDeletesTheAuthenticatedUsersAccount(t *testing.T) {
	deleter := &recordingUserDeleter{}

	recorder := sendDeleteUser(t, sendDeleteUserParams{deleter: deleter})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", recorder.Body.String())
	}
	if len(deleter.receivedCalls) != 1 {
		t.Fatalf("DeleteUser was called %d times, want 1", len(deleter.receivedCalls))
	}
	call := deleter.receivedCalls[0]
	if call.UserID != deletingUserID {
		t.Errorf("call = %+v, want user %s", call, deletingUserID)
	}
}

func TestDeleteUserFailsWhenTheUserIDIsMissingFromContext(t *testing.T) {
	silenceHandlerLogger(t)
	deleter := &recordingUserDeleter{}

	recorder := sendDeleteUser(t, sendDeleteUserParams{
		deleter:   deleter,
		skipGuard: true,
	})

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if len(deleter.receivedCalls) != 0 {
		t.Errorf("DeleteUser was called %d times, want 0", len(deleter.receivedCalls))
	}
}

func TestDeleteUserFailsWhenTheDeletionFails(t *testing.T) {
	cases := []struct {
		name        string
		failure     error
		wantStatus  int
		wantCode    handlers.ErrorCode
		wantMessage string
	}{
		{
			name:        "the account no longer exists",
			failure:     usererrors.ErrUserNotFound{ID: deletingUserID},
			wantStatus:  http.StatusNotFound,
			wantCode:    handlers.ErrorCodeUserNotFound,
			wantMessage: userNotFoundText,
		},
		{
			name:        "something unexpected broke",
			failure:     usererrors.ErrSoftDeleteUserFailed{Cause: errors.New("connection reset")},
			wantStatus:  http.StatusInternalServerError,
			wantCode:    handlers.ErrorCodeInternal,
			wantMessage: unexpectedDeletionFailureText,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			logs := captureHandlerLogger(t)
			deleter := &recordingUserDeleter{failure: testCase.failure}

			recorder := sendDeleteUser(t, sendDeleteUserParams{deleter: deleter})

			if recorder.Code != testCase.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, testCase.wantStatus)
			}
			body := decodeBody(t, recorder.Result())
			if body["code"] != string(testCase.wantCode) {
				t.Errorf("code = %v, want %q", body["code"], testCase.wantCode)
			}
			if body["message"] != testCase.wantMessage {
				t.Errorf("message = %v, want %q", body["message"], testCase.wantMessage)
			}
			if loggedError := decodeLoggedError(t, logs); loggedError != testCase.failure.Error() {
				t.Errorf("logged error = %q, want the cause %q", loggedError, testCase.failure.Error())
			}
		})
	}
}
