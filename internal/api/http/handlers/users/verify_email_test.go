package users

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	models "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

const presentedVerificationToken = "Kq7wL2mZ9xR4tV6yB1nH8pD3sG5jF0aC"

const verifyEmailPath = "/api/v1/users/verify-email"

var verifyingUserID = uuid.MustParse("6f1c9f2e-4a1b-4c3d-9e8f-0a1b2c3d4e5f")

type recordingEmailVerifier struct {
	failure       error
	receivedCalls []usecases.VerifyEmailParams
}

func (v *recordingEmailVerifier) verifyEmail(
	_ context.Context,
	params usecases.VerifyEmailParams,
) error {
	v.receivedCalls = append(v.receivedCalls, params)

	return v.failure
}

func authenticateForTest(userID uuid.UUID) func(http.Handler) http.Handler {
	return middleware.RequireAuthenticatedUser(middleware.RequireAuthenticatedUserDependencies{
		ParseAccessToken: func(string) (jwt.AccessTokenClaims, error) {
			return jwt.AccessTokenClaims{UserID: userID}, nil
		},
		FindUserByID: func(_ context.Context, id uuid.UUID) (models.User, error) {
			return models.User{ID: id, Email: "accountant@example.com"}, nil
		},
	})
}

type postVerifyEmailParams struct {
	verifier  *recordingEmailVerifier
	query     string
	skipGuard bool
}

func postVerifyEmail(t *testing.T, params postVerifyEmailParams) *httptest.ResponseRecorder {
	t.Helper()

	handler := VerifyEmail(VerifyEmailDependencies{
		VerifyEmail: params.verifier.verifyEmail,
	})

	var served http.Handler = handler
	if !params.skipGuard {
		served = authenticateForTest(verifyingUserID)(handler)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, verifyEmailPath+params.query, nil)
	request.Header.Set("Authorization", "Bearer header.payload.signature")

	served.ServeHTTP(recorder, request)

	return recorder
}

func TestVerifyEmailRedeemsTheLinkForTheAuthenticatedUser(t *testing.T) {
	verifier := &recordingEmailVerifier{}

	recorder := postVerifyEmail(t, postVerifyEmailParams{
		verifier: verifier,
		query:    "?token=" + presentedVerificationToken,
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if len(verifier.receivedCalls) != 1 {
		t.Fatalf("VerifyEmail was called %d times, want 1", len(verifier.receivedCalls))
	}

	call := verifier.receivedCalls[0]
	if call.VerificationToken != presentedVerificationToken {
		t.Errorf("VerificationToken = %q, want %q", call.VerificationToken, presentedVerificationToken)
	}
	if call.UserID != verifyingUserID {
		t.Errorf("UserID = %s, want %s", call.UserID, verifyingUserID)
	}
}

func TestVerifyEmailFailsWhenTheUserIDIsMissingFromContext(t *testing.T) {
	silenceHandlerLogger(t)

	verifier := &recordingEmailVerifier{}

	recorder := postVerifyEmail(t, postVerifyEmailParams{
		verifier:  verifier,
		query:     "?token=" + presentedVerificationToken,
		skipGuard: true,
	})

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if len(verifier.receivedCalls) != 0 {
		t.Errorf("VerifyEmail was called %d times, want 0", len(verifier.receivedCalls))
	}
}

func TestVerifyEmailFailsWhenTheVerificationTokenIsMissing(t *testing.T) {
	verifier := &recordingEmailVerifier{}

	recorder := postVerifyEmail(t, postVerifyEmailParams{verifier: verifier})

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if len(verifier.receivedCalls) != 0 {
		t.Errorf("VerifyEmail was called %d times, want 0", len(verifier.receivedCalls))
	}
}

func captureHandlerLogger(t *testing.T) *bytes.Buffer {
	t.Helper()

	logs := &bytes.Buffer{}
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	return logs
}

func decodeLoggedError(t *testing.T, logs *bytes.Buffer) string {
	t.Helper()

	entry := map[string]any{}
	if err := json.NewDecoder(logs).Decode(&entry); err != nil {
		t.Fatalf("decoding the log entry: %v", err)
	}

	loggedError, _ := entry["error"].(string)

	return loggedError
}

func TestVerifyEmailFailsWhenTheVerificationFails(t *testing.T) {
	cases := []struct {
		name        string
		failure     error
		wantStatus  int
		wantCode    handlers.ErrorCode
		wantMessage string
	}{
		{
			name:        "the link is invalid",
			failure:     usererrors.ErrVerificationLinkInvalid{},
			wantStatus:  http.StatusBadRequest,
			wantCode:    handlers.ErrorCodeVerificationLinkInvalid,
			wantMessage: invalidVerificationLinkText,
		},
		{
			name: "the link was issued for another purpose",
			failure: challenge.ErrChallengePurposeMismatch{
				Want: challenge.PurposeEmailVerification,
				Got:  challenge.PasswordReset,
			},
			wantStatus:  http.StatusBadRequest,
			wantCode:    handlers.ErrorCodeVerificationLinkInvalid,
			wantMessage: invalidVerificationLinkText,
		},
		{
			name: "the link belongs to another user",
			failure: challenge.ErrChallengeOwnedByAnotherUser{
				ChallengeID: uuid.MustParse("3f8b1c94-9a2d-4f61-8c07-5b1e2d7a6c30"),
			},
			wantStatus:  http.StatusForbidden,
			wantCode:    handlers.ErrorCodeVerificationLinkNotOwned,
			wantMessage: verificationLinkNotYoursText,
		},
		{
			name:        "the new address was claimed since the request",
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
			wantMessage: unexpectedVerificationFailureText,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			logs := captureHandlerLogger(t)

			recorder := postVerifyEmail(t, postVerifyEmailParams{
				verifier: &recordingEmailVerifier{failure: testCase.failure},
				query:    "?token=" + presentedVerificationToken,
			})

			if recorder.Code != testCase.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, testCase.wantStatus)
			}

			body := decodeBody(t, recorder.Result())
			if code, _ := body["code"].(string); code != string(testCase.wantCode) {
				t.Errorf(`body["code"] = %q, want %q`, code, testCase.wantCode)
			}
			if message, _ := body["message"].(string); message != testCase.wantMessage {
				t.Errorf(`body["message"] = %q, want %q`, message, testCase.wantMessage)
			}
			if loggedError := decodeLoggedError(t, logs); loggedError != testCase.failure.Error() {
				t.Errorf(`logged "error" = %q, want the reason %q`, loggedError, testCase.failure.Error())
			}
		})
	}
}
