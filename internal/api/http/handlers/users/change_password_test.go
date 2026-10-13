package users

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

const changePasswordPath = "/api/v1/users/change-password"

const changePasswordBodyForTest = `{"currentPassword":"correct-horse","newPassword":"new-ledger-password"}`

type recordingPasswordChanger struct {
	failure       error
	receivedCalls []usecases.ChangePasswordParams
}

func (r *recordingPasswordChanger) changePassword(_ context.Context, params usecases.ChangePasswordParams) error {
	r.receivedCalls = append(r.receivedCalls, params)

	return r.failure
}

type postChangePasswordParams struct {
	changer   *recordingPasswordChanger
	body      string
	skipGuard bool
}

func postChangePassword(t *testing.T, params postChangePasswordParams) *http.Response {
	t.Helper()

	handler := ChangePassword(ChangePasswordDependencies{
		ChangePassword: params.changer.changePassword,
	})

	var served http.Handler = handler
	if !params.skipGuard {
		served = authenticateForTest(verifyingUserID)(handler)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, changePasswordPath, strings.NewReader(params.body))
	request.Header.Set("Authorization", "Bearer header.payload.signature")
	request.Header.Set("Content-Type", "application/json")

	served.ServeHTTP(recorder, request)

	response := recorder.Result()
	t.Cleanup(func() { response.Body.Close() })

	return response
}

func TestChangePasswordHandsBothPasswordsToTheUseCase(t *testing.T) {
	changer := &recordingPasswordChanger{}

	response := postChangePassword(t, postChangePasswordParams{changer: changer, body: changePasswordBodyForTest})

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}
	if len(changer.receivedCalls) != 1 {
		t.Fatalf("ChangePassword was called %d times, want 1", len(changer.receivedCalls))
	}

	want := usecases.ChangePasswordParams{
		UserID:          verifyingUserID,
		CurrentPassword: "correct-horse",
		NewPassword:     "new-ledger-password",
	}
	if changer.receivedCalls[0] != want {
		t.Errorf("params = %+v, want %+v", changer.receivedCalls[0], want)
	}
}

func TestChangePasswordFailsWhenTheUserIDIsMissingFromContext(t *testing.T) {
	silenceHandlerLogger(t)

	changer := &recordingPasswordChanger{}

	response := postChangePassword(t, postChangePasswordParams{
		changer:   changer,
		body:      changePasswordBodyForTest,
		skipGuard: true,
	})

	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusInternalServerError)
	}
	if len(changer.receivedCalls) != 0 {
		t.Errorf("ChangePassword was called %d times, want 0", len(changer.receivedCalls))
	}
}

func TestChangePasswordRejectsABadRequestBeforeTheUseCase(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantCode handlers.ErrorCode
	}{
		{name: "malformed json", body: `{"currentPassword":`, wantCode: handlers.ErrorCodeRequestBodyInvalid},
		{name: "missing current password", body: `{"newPassword":"new-ledger-password"}`,
			wantCode: handlers.ErrorCodePasswordMissing},
		{name: "new password too short", body: `{"currentPassword":"correct-horse","newPassword":"short"}`,
			wantCode: handlers.ErrorCodePasswordTooShort},
		{name: "new password too long",
			body:     `{"currentPassword":"correct-horse","newPassword":"` + strings.Repeat("a", 65) + `"}`,
			wantCode: handlers.ErrorCodePasswordTooLong},
		{name: "new password with whitespace",
			body:     `{"currentPassword":"correct-horse","newPassword":"new ledger password"}`,
			wantCode: handlers.ErrorCodePasswordContainsWhitespace},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			changer := &recordingPasswordChanger{}

			response := postChangePassword(t, postChangePasswordParams{changer: changer, body: testCase.body})

			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusBadRequest)
			}
			body := decodeBody(t, response)
			if code, _ := body["code"].(string); code != string(testCase.wantCode) {
				t.Errorf(`body["code"] = %q, want %q`, code, testCase.wantCode)
			}
			if len(changer.receivedCalls) != 0 {
				t.Errorf("ChangePassword was called %d times, want 0", len(changer.receivedCalls))
			}
		})
	}
}

func TestChangePasswordMapsUseCaseFailures(t *testing.T) {
	cases := []struct {
		name        string
		failure     error
		wantStatus  int
		wantCode    handlers.ErrorCode
		wantMessage string
	}{
		{
			name:        "current password does not match",
			failure:     usererrors.ErrCurrentPasswordIncorrect{},
			wantStatus:  http.StatusForbidden,
			wantCode:    handlers.ErrorCodeCurrentPasswordIncorrect,
			wantMessage: usererrors.ErrCurrentPasswordIncorrect{}.Error(),
		},
		{
			name:        "new password equals the current one",
			failure:     usererrors.ErrPasswordUnchanged{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    handlers.ErrorCodePasswordUnchanged,
			wantMessage: usererrors.ErrPasswordUnchanged{}.Error(),
		},
		{
			name:        "authenticated user vanished",
			failure:     usererrors.ErrUserNotFound{ID: verifyingUserID},
			wantStatus:  http.StatusInternalServerError,
			wantCode:    handlers.ErrorCodeInternal,
			wantMessage: unexpectedPasswordChangeFailureText,
		},
		{
			name:        "something unexpected broke",
			failure:     errors.New("connection reset"),
			wantStatus:  http.StatusInternalServerError,
			wantCode:    handlers.ErrorCodeInternal,
			wantMessage: unexpectedPasswordChangeFailureText,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			silenceHandlerLogger(t)

			response := postChangePassword(t, postChangePasswordParams{
				changer: &recordingPasswordChanger{failure: testCase.failure},
				body:    changePasswordBodyForTest,
			})

			if response.StatusCode != testCase.wantStatus {
				t.Fatalf("status = %d, want %d", response.StatusCode, testCase.wantStatus)
			}

			body := decodeBody(t, response)
			if code, _ := body["code"].(string); code != string(testCase.wantCode) {
				t.Errorf(`body["code"] = %q, want %q`, code, testCase.wantCode)
			}
			if message, _ := body["message"].(string); message != testCase.wantMessage {
				t.Errorf(`body["message"] = %q, want %q`, message, testCase.wantMessage)
			}
		})
	}
}
