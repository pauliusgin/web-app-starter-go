package messages

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
	"github.com/pauliusgin/web_app_starter-be/internal/events/eventstest"
)

const passwordResetEmailBodyForTest = `{"email":"accountant@example.com"}`

var requestArrivedAt = time.Date(2026, time.November, 2, 9, 0, 0, 0, time.UTC)

// frozenResponseClock never advances and records each wait instead of sleeping.
type frozenResponseClock struct {
	waited []time.Duration
}

func (c *frozenResponseClock) clock() handlers.ResponseClock {
	return handlers.ResponseClock{
		Now:  func() time.Time { return requestArrivedAt },
		Wait: func(_ context.Context, duration time.Duration) { c.waited = append(c.waited, duration) },
	}
}

func (c *frozenResponseClock) assertWaited(t *testing.T, want ...time.Duration) {
	t.Helper()

	if len(c.waited) != len(want) {
		t.Fatalf("waited = %v, want %v", c.waited, want)
	}
	for index, duration := range want {
		if c.waited[index] != duration {
			t.Errorf("waited[%d] = %s, want %s", index, c.waited[index], duration)
		}
	}
}

// passwordResetStores is the state a password reset email request reads and changes.
type passwordResetStores struct {
	users      *repositories.InMemoryUserRepository
	challenges *challenge.InMemoryChallengeRepository
	clock      *frozenResponseClock
}

func newPasswordResetStores() *passwordResetStores {
	return &passwordResetStores{
		users:      repositories.NewInMemoryUserRepository(),
		challenges: challenge.NewInMemoryChallengeRepository(),
		clock:      &frozenResponseClock{},
	}
}

func (s *passwordResetStores) seedPasswordUser() uuid.UUID {
	userID := uuid.New()
	s.users.Users[userID] = repositories.InMemoryUser{
		User:         models.User{ID: userID, Email: testEmail},
		PasswordHash: "$argon2id$seeded",
	}

	return userID
}

func (s *passwordResetStores) sendPasswordResetEmail(t *testing.T, body string) *http.Response {
	t.Helper()

	handler := SendPasswordResetEmail(SendPasswordResetEmailDependencies{
		RequestPasswordReset: func(ctx context.Context, params usecases.RequestPasswordResetParams) error {
			return usecases.RequestPasswordReset(ctx, usecases.RequestPasswordResetDependencies{
				UserWriteRepository: s.users,
				EnsureChallengeIssuanceAllowed: func(context.Context, challenge.EnsureIssuanceAllowedParams) error {
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
		Clock: s.clock.clock(),
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/messages/password-reset-email", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")

	handler.ServeHTTP(recorder, request)

	response := recorder.Result()
	t.Cleanup(func() { response.Body.Close() })

	return response
}

func TestSendPasswordResetEmailSendsAResetLinkToThePasswordAccount(t *testing.T) {
	stores := newPasswordResetStores()
	userID := stores.seedPasswordUser()

	response := stores.sendPasswordResetEmail(t, `{"email":" Accountant@Example.com "}`)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if body, _ := io.ReadAll(response.Body); len(body) != 0 {
		t.Errorf("body = %q, want empty", body)
	}
	links := stores.challenges.Challenges
	if len(links) != 1 || links[0].UserID != userID || links[0].Purpose != challenge.PurposePasswordReset {
		t.Errorf("stored links = %+v, want one password reset link for %s", links, userID)
	}
	stores.clock.assertWaited(t, passwordResetResponseFloor)
}

// An unknown email must look exactly like a known one: same status, same empty body, same padded timing.
func TestSendPasswordResetEmailAnswersAnUnknownEmailLikeAKnownOne(t *testing.T) {
	stores := newPasswordResetStores()

	response := stores.sendPasswordResetEmail(t, passwordResetEmailBodyForTest)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if body, _ := io.ReadAll(response.Body); len(body) != 0 {
		t.Errorf("body = %q, want empty", body)
	}
	if len(stores.challenges.Challenges) != 0 {
		t.Errorf("stored links = %+v, want none", stores.challenges.Challenges)
	}
	stores.clock.assertWaited(t, passwordResetResponseFloor)
}

func TestSendPasswordResetEmailPadsAFailedLookupToTheFloor(t *testing.T) {
	silenceHandlerLogger(t)
	stores := newPasswordResetStores()
	stores.seedPasswordUser()
	stores.users.LookupFailure = usererrors.ErrFindUserFailed{Cause: io.ErrUnexpectedEOF}

	response := stores.sendPasswordResetEmail(t, passwordResetEmailBodyForTest)

	assertErrorCode(t, response, http.StatusInternalServerError, handlers.ErrorCodeInternal)
	stores.clock.assertWaited(t, passwordResetResponseFloor)
}

func TestSendPasswordResetEmailRejectsAMalformedRequestWithoutWaiting(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantCode handlers.ErrorCode
	}{
		{name: "malformed json", body: `{"email":`, wantCode: handlers.ErrorCodeRequestBodyInvalid},
		{name: "invalid email", body: `{"email":"bob"}`, wantCode: handlers.ErrorCodeEmailInvalid},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			stores := newPasswordResetStores()
			stores.seedPasswordUser()

			response := stores.sendPasswordResetEmail(t, testCase.body)

			assertErrorCode(t, response, http.StatusBadRequest, testCase.wantCode)
			if len(stores.challenges.Challenges) != 0 {
				t.Errorf("stored links = %+v, want none", stores.challenges.Challenges)
			}
			stores.clock.assertWaited(t)
		})
	}
}
