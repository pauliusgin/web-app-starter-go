package messaging

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/gateways/email"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/usecases"
	"github.com/pauliusgin/web_app_starter-be/internal/events/auth"
)

const testFrontendBaseURL = "https://app.example.com/web_app_starter"

const testVerificationToken = "abc123"

const testChallengeTarget = "accountant@example.com"

var testUserID = uuid.MustParse("1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15")

var testChallengeID = uuid.MustParse("0b3e2c1d-7f6a-4e5b-8c9d-1a2b3c4d5e6f")

func silenceDefaultLogger(t *testing.T) {
	t.Helper()

	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
}

func testAuthChallengePayload(purpose challenge.Purpose) auth.AuthChallengeGeneratedData {
	return auth.AuthChallengeGeneratedData{
		ChallengeID: testChallengeID,
		UserID:      testUserID,
		TargetType:  string(challenge.TargetTypeEmail),
		Target:      testChallengeTarget,
		Purpose:     string(purpose),
		Token:       testVerificationToken,
	}
}

func newTestVerificationEmailHandler(
	gateway *email.InMemoryGateway,
	messageRepository *repositories.InMemoryMessageWriteRepository,
) func(context.Context, auth.AuthChallengeGeneratedData) error {
	dependencies := usecases.SendVerificationEmailDependencies{
		Gateway:           gateway,
		MessageRepository: messageRepository,
		FrontendBaseURL:   testFrontendBaseURL,
	}

	return VerificationEmailOnChallenge(VerificationEmailDependencies{
		SendVerificationEmail: func(ctx context.Context, params usecases.SendVerificationEmailParams) error {
			return usecases.SendVerificationEmail(ctx, dependencies, params)
		},
	})
}

func TestVerificationEmailHandlerSendsOneLinkBuiltFromTheConfiguredFrontend(t *testing.T) {
	silenceDefaultLogger(t)

	gateway := email.NewInMemoryGateway()
	messageRepository := repositories.NewInMemoryMessageWriteRepository()
	sendVerificationEmail := newTestVerificationEmailHandler(gateway, messageRepository)

	err := sendVerificationEmail(context.Background(),
		testAuthChallengePayload(challenge.PurposeEmailVerification))
	if err != nil {
		t.Fatalf("sendVerificationEmail: %v", err)
	}

	message, ok := gateway.OnlySentMessage()
	if !ok {
		t.Fatalf("sent messages = %d, want 1", len(gateway.Sent))
	}

	if message.RecipientEmail != testChallengeTarget {
		t.Errorf("RecipientEmail = %q, want %q", message.RecipientEmail, testChallengeTarget)
	}
	if !strings.HasPrefix(message.VerificationLink, testFrontendBaseURL) {
		t.Errorf("VerificationLink = %q, want it built from %q",
			message.VerificationLink, testFrontendBaseURL)
	}

	verificationLink, err := url.Parse(message.VerificationLink)
	if err != nil {
		t.Fatalf("parsing the built url %q: %v", message.VerificationLink, err)
	}
	if verificationLink.Query().Get("token") != testVerificationToken {
		t.Errorf("token query = %q, want %q",
			verificationLink.Query().Get("token"), testVerificationToken)
	}

	loggedMessage, ok := messageRepository.OnlySavedMessage()
	if !ok {
		t.Fatalf("saved messages = %d, want 1", len(messageRepository.Saved))
	}
	if loggedMessage.RecipientID != testUserID {
		t.Errorf("logged RecipientID = %s, want the challenge's user %s", loggedMessage.RecipientID, testUserID)
	}
	if !strings.Contains(string(loggedMessage.Data), testChallengeID.String()) {
		t.Errorf("logged Data = %s, want it to reference challenge %s", loggedMessage.Data, testChallengeID)
	}
}

// Every auth challenge is announced under one routing key. Mailing a verification link for a
// password reset would put a reset token behind a /verify-email URL.
func TestVerificationEmailHandlerIgnoresChallengesWithoutAnEmailedLink(t *testing.T) {
	silenceDefaultLogger(t)

	otherPurposes := []challenge.Purpose{
		challenge.PurposePhoneVerification,
		challenge.PasswordReset,
	}

	for _, purpose := range otherPurposes {
		t.Run(string(purpose), func(t *testing.T) {
			gateway := email.NewInMemoryGateway()
			messageRepository := repositories.NewInMemoryMessageWriteRepository()
			sendVerificationEmail := newTestVerificationEmailHandler(gateway, messageRepository)

			err := sendVerificationEmail(context.Background(), testAuthChallengePayload(purpose))
			if err != nil {
				t.Fatalf("sendVerificationEmail = %v, want an ignored challenge to be no error", err)
			}
			if len(gateway.Sent) != 0 {
				t.Errorf("sent messages = %d, want none for %q", len(gateway.Sent), purpose)
			}
			if len(messageRepository.Saved) != 0 {
				t.Errorf("saved messages = %d, want none logged for %q", len(messageRepository.Saved), purpose)
			}
		})
	}
}

func TestVerificationEmailHandlerReportsAFailingGateway(t *testing.T) {
	silenceDefaultLogger(t)

	providerFailure := errors.New("brevo refused the verification email: status 400")
	gateway := email.NewInMemoryGateway()
	gateway.SendFailure = providerFailure

	sendVerificationEmail := newTestVerificationEmailHandler(gateway, repositories.NewInMemoryMessageWriteRepository())

	err := sendVerificationEmail(context.Background(),
		testAuthChallengePayload(challenge.PurposeEmailVerification))

	if err == nil {
		t.Fatal("sendVerificationEmail = nil, want the gateway's failure reported")
	}
	if !errors.Is(err, providerFailure) {
		t.Errorf("error = %v, want it to unwrap to the gateway's own failure", err)
	}
}

func TestVerificationEmailHandlerIsNilWithoutASender(t *testing.T) {
	handler := VerificationEmailOnChallenge(VerificationEmailDependencies{})

	if handler != nil {
		t.Error("handler = non-nil, want none without a use case to call")
	}
}
