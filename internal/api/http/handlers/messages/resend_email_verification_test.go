package messages

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
	"github.com/pauliusgin/web_app_starter-be/internal/events/eventstest"
)

const testEmail = "accountant@example.com"

// verifiedAt is when a seeded user proved their email; any set instant marks the address verified.
var verifiedAt = time.Date(2026, time.October, 1, 9, 0, 0, 0, time.UTC)

// resendStores is the state a resend request reads and changes; challenges is the auth_challenges table.
type resendStores struct {
	users       *repositories.InMemoryUserRepository
	challenges  *challenge.InMemoryChallengeRepository
	rateLimited bool
}

func newResendStores() *resendStores {
	return &resendStores{
		users:      repositories.NewInMemoryUserRepository(),
		challenges: challenge.NewInMemoryChallengeRepository(),
	}
}

func (s *resendStores) seedUser(verifiedAt time.Time) uuid.UUID {
	userID := uuid.New()
	s.users.Users[userID] = repositories.InMemoryUser{User: models.User{
		ID:              userID,
		Email:           testEmail,
		EmailVerifiedAt: verifiedAt,
	}}

	return userID
}

func (s *resendStores) handler() http.Handler {
	return ResendEmailVerification(ResendEmailVerificationDependencies{
		ResendEmailVerification: func(ctx context.Context, params usecases.ResendEmailVerificationParams) error {
			return usecases.ResendEmailVerification(ctx, usecases.ResendEmailVerificationDependencies{
				UserWriteRepository: s.users,
				EnsureChallengeIssuanceAllowed: func(context.Context, challenge.EnsureIssuanceAllowedParams) error {
					if s.rateLimited {
						return challenge.ErrIssuanceRateLimited{RetryAfter: 20 * time.Second}
					}
					return nil
				},
				GenerateAuthChallenge: func(ctx context.Context, params challenge.GenerateParams) (challenge.GenerateResult, error) {
					return challenge.Generate(ctx, challenge.GenerateDependencies{
						WriteRepository: s.challenges,
						EventDispatcher: (&eventstest.CapturedEvents{}).Dispatcher(),
					}, params)
				},
			}, params)
		},
	})
}

// sendResend posts as asUserID; uuid.Nil sends it past no guard, as an unprotected route would.
func (s *resendStores) sendResend(t *testing.T, asUserID uuid.UUID) *http.Response {
	t.Helper()

	served := s.handler()
	if asUserID != uuid.Nil {
		served = middleware.RequireAuthenticatedUser(middleware.RequireAuthenticatedUserDependencies{
			ParseAccessToken: func(string) (jwt.AccessTokenClaims, error) {
				return jwt.AccessTokenClaims{UserID: asUserID}, nil
			},
			FindUserByID: func(_ context.Context, id uuid.UUID) (models.User, error) {
				return models.User{ID: id, Email: testEmail}, nil
			},
		})(served)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/messages/verification-email", nil)
	request.Header.Set("Authorization", "Bearer header.payload.signature")

	served.ServeHTTP(recorder, request)

	response := recorder.Result()
	t.Cleanup(func() { response.Body.Close() })

	return response
}

func assertErrorCode(t *testing.T, response *http.Response, wantStatus int, wantCode handlers.ErrorCode) {
	t.Helper()

	if response.StatusCode != wantStatus {
		t.Fatalf("status = %d, want %d", response.StatusCode, wantStatus)
	}
	body := map[string]any{}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decoding the response body: %v", err)
	}
	if body["code"] != string(wantCode) {
		t.Errorf(`body["code"] = %v, want %q`, body["code"], wantCode)
	}
}

func silenceHandlerLogger(t *testing.T) {
	t.Helper()

	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
}

func TestResendEmailVerificationSendsALinkToTheAuthenticatedUsersEmail(t *testing.T) {
	stores := newResendStores()
	userID := stores.seedUser(time.Time{})

	response := stores.sendResend(t, userID)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if body, _ := io.ReadAll(response.Body); len(body) != 0 {
		t.Errorf("body = %q, want empty", body)
	}
	links := stores.challenges.Challenges
	if len(links) != 1 || links[0].UserID != userID || links[0].Target != testEmail || links[0].IsUsed() {
		t.Errorf("stored links = %+v, want one unused link to %q for %s", links, testEmail, userID)
	}
}

func TestResendEmailVerificationRejectsWithoutSendingALink(t *testing.T) {
	cases := []struct {
		name       string
		arrange    func(stores *resendStores) uuid.UUID
		wantStatus int
		wantCode   handlers.ErrorCode
	}{
		{
			name:       "email already verified",
			arrange:    func(stores *resendStores) uuid.UUID { return stores.seedUser(verifiedAt) },
			wantStatus: http.StatusConflict,
			wantCode:   handlers.ErrorCodeEmailAlreadyVerified,
		},
		{
			name:       "account gone",
			arrange:    func(*resendStores) uuid.UUID { return uuid.New() },
			wantStatus: http.StatusNotFound,
			wantCode:   handlers.ErrorCodeUserNotFound,
		},
		{
			name: "lookup fails",
			arrange: func(stores *resendStores) uuid.UUID {
				stores.users.LookupFailure = usererrors.ErrFindUserFailed{Cause: io.ErrUnexpectedEOF}
				return stores.seedUser(time.Time{})
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   handlers.ErrorCodeInternal,
		},
		{
			name:       "route unprotected",
			arrange:    func(stores *resendStores) uuid.UUID { stores.seedUser(time.Time{}); return uuid.Nil },
			wantStatus: http.StatusInternalServerError,
			wantCode:   handlers.ErrorCodeInternal,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			silenceHandlerLogger(t)
			stores := newResendStores()

			response := stores.sendResend(t, testCase.arrange(stores))

			assertErrorCode(t, response, testCase.wantStatus, testCase.wantCode)
			if len(stores.challenges.Challenges) != 0 {
				t.Errorf("stored links = %+v, want none", stores.challenges.Challenges)
			}
		})
	}
}

func TestResendEmailVerificationAnswersARateLimitWithTheWait(t *testing.T) {
	silenceHandlerLogger(t)
	stores := newResendStores()
	userID := stores.seedUser(time.Time{})
	stores.rateLimited = true

	response := stores.sendResend(t, userID)

	if retryAfter := response.Header.Get("Retry-After"); retryAfter != "20" {
		t.Errorf("Retry-After = %q, want %q", retryAfter, "20")
	}
	assertErrorCode(t, response, http.StatusTooManyRequests, handlers.ErrorCodeVerificationEmailRateLimited)
}
