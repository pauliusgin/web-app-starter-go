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

const requestEmailChangePath = "/api/v1/users/change-email"

const requestEmailChangeBodyForTest = `{"newEmail":"  Bookkeeper@Example.com "}`

type recordingEmailChangeRequester struct {
	failure       error
	receivedCalls []usecases.RequestEmailChangeParams
}

func (r *recordingEmailChangeRequester) requestEmailChange(
	_ context.Context,
	params usecases.RequestEmailChangeParams,
) error {
	r.receivedCalls = append(r.receivedCalls, params)

	return r.failure
}

type postRequestEmailChangeParams struct {
	requester *recordingEmailChangeRequester
	body      string
	skipGuard bool
}

func postRequestEmailChange(t *testing.T, params postRequestEmailChangeParams) *http.Response {
	t.Helper()

	handler := RequestEmailChange(RequestEmailChangeDependencies{
		RequestEmailChange: params.requester.requestEmailChange,
	})

	var served http.Handler = handler
	if !params.skipGuard {
		served = authenticateForTest(verifyingUserID)(handler)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, requestEmailChangePath, strings.NewReader(params.body))
	request.Header.Set("Authorization", "Bearer header.payload.signature")
	request.Header.Set("Content-Type", "application/json")

	served.ServeHTTP(recorder, request)

	response := recorder.Result()
	t.Cleanup(func() { response.Body.Close() })

	return response
}

func TestRequestEmailChangeHandsTheNormalizedEmailToTheUseCase(t *testing.T) {
	requester := &recordingEmailChangeRequester{}

	response := postRequestEmailChange(t, postRequestEmailChangeParams{
		requester: requester,
		body:      requestEmailChangeBodyForTest,
	})

	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusAccepted)
	}
	if len(requester.receivedCalls) != 1 {
		t.Fatalf("RequestEmailChange was called %d times, want 1", len(requester.receivedCalls))
	}

	want := usecases.RequestEmailChangeParams{
		UserID:   verifyingUserID,
		NewEmail: "bookkeeper@example.com",
	}
	if requester.receivedCalls[0] != want {
		t.Errorf("params = %+v, want %+v", requester.receivedCalls[0], want)
	}
}

func TestRequestEmailChangeFailsWhenTheUserIDIsMissingFromContext(t *testing.T) {
	silenceHandlerLogger(t)

	requester := &recordingEmailChangeRequester{}

	response := postRequestEmailChange(t, postRequestEmailChangeParams{
		requester: requester,
		body:      requestEmailChangeBodyForTest,
		skipGuard: true,
	})

	if response.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusInternalServerError)
	}
	if len(requester.receivedCalls) != 0 {
		t.Errorf("RequestEmailChange was called %d times, want 0", len(requester.receivedCalls))
	}
}

func TestRequestEmailChangeRejectsABadRequestBeforeTheUseCase(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantCode handlers.ErrorCode
	}{
		{name: "malformed json", body: `{"newEmail":`, wantCode: handlers.ErrorCodeRequestBodyInvalid},
		{name: "invalid email", body: `{"newEmail":"bob"}`, wantCode: handlers.ErrorCodeEmailInvalid},
		{name: "missing email", body: `{}`, wantCode: handlers.ErrorCodeEmailInvalid},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			requester := &recordingEmailChangeRequester{}

			response := postRequestEmailChange(t, postRequestEmailChangeParams{requester: requester, body: testCase.body})

			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusBadRequest)
			}
			body := decodeBody(t, response)
			if code, _ := body["code"].(string); code != string(testCase.wantCode) {
				t.Errorf(`body["code"] = %q, want %q`, code, testCase.wantCode)
			}
			if len(requester.receivedCalls) != 0 {
				t.Errorf("RequestEmailChange was called %d times, want 0", len(requester.receivedCalls))
			}
		})
	}
}

func TestRequestEmailChangeMapsUseCaseFailures(t *testing.T) {
	cases := []struct {
		name        string
		failure     error
		wantStatus  int
		wantCode    handlers.ErrorCode
		wantMessage string
	}{
		{
			name:        "same email as now",
			failure:     usererrors.ErrEmailUnchanged{Email: "bookkeeper@example.com"},
			wantStatus:  http.StatusBadRequest,
			wantCode:    handlers.ErrorCodeEmailUnchanged,
			wantMessage: usererrors.ErrEmailUnchanged{Email: "bookkeeper@example.com"}.Error(),
		},
		{
			name:        "email held by another account",
			failure:     usererrors.ErrEmailTaken{Email: "bookkeeper@example.com"},
			wantStatus:  http.StatusConflict,
			wantCode:    handlers.ErrorCodeEmailTaken,
			wantMessage: usererrors.ErrEmailTaken{Email: "bookkeeper@example.com"}.Error(),
		},
		{
			name:        "something unexpected broke",
			failure:     errors.New("connection reset"),
			wantStatus:  http.StatusInternalServerError,
			wantCode:    handlers.ErrorCodeInternal,
			wantMessage: unexpectedEmailChangeRequestFailureText,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			silenceHandlerLogger(t)

			response := postRequestEmailChange(t, postRequestEmailChangeParams{
				requester: &recordingEmailChangeRequester{failure: testCase.failure},
				body:      requestEmailChangeBodyForTest,
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
