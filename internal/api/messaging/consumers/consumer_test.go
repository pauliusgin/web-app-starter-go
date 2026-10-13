package consumers

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/messaging/handlers/messaging"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/gateways/email"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/usecases"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/auth"
)

const testFrontendBaseURL = "https://app.example.com/web_app_starter"

const testVerificationToken = "abc123"

const testChallengeTarget = "accountant@example.com"

func silenceDefaultLogger(t *testing.T) {
	t.Helper()

	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
}

func newTestAuthChallengeEvent(t *testing.T, purpose challenge.Purpose) events.NewEvent {
	t.Helper()

	generated, err := auth.NewAuthChallengeGenerated(auth.NewAuthChallengeParams{
		ChallengeID: uuid.MustParse("6f1b1a3e-1d2c-4a5b-8c7d-9e0f1a2b3c4d"),
		UserID:      uuid.MustParse("11111111-2222-3333-4444-555555555555"),
		TargetType:  string(challenge.TargetTypeEmail),
		Target:      testChallengeTarget,
		Purpose:     string(purpose),
		Token:       testVerificationToken,
		ExpiresAt:   time.Date(2026, time.September, 11, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("NewAuthChallengeGenerated: %v", err)
	}

	return generated
}

func publishedBody(event events.NewEvent) []byte {
	return event.Data
}

func newTestRegistry(gateway *email.InMemoryGateway) Registry {
	dependencies := usecases.SendVerificationEmailDependencies{
		Gateway:         gateway,
		FrontendBaseURL: testFrontendBaseURL,
	}

	return BuildRegistry(RegistryDependencies{
		VerificationEmail: messaging.VerificationEmailDependencies{
			SendVerificationEmail: func(ctx context.Context, params usecases.SendVerificationEmailParams) error {
				return usecases.SendVerificationEmail(ctx, dependencies, params)
			},
		},
	})
}

func onlyConsumer(t *testing.T, registry Registry) Consumer {
	t.Helper()

	if len(registry) != 1 {
		t.Fatalf("registry = %d consumers, want 1", len(registry))
	}

	return registry[0]
}

func TestDeliveringAPublishedChallengeReachesTheGateway(t *testing.T) {
	silenceDefaultLogger(t)

	gateway := email.NewInMemoryGateway()
	consumer := onlyConsumer(t, newTestRegistry(gateway))

	event := newTestAuthChallengeEvent(t, challenge.PurposeEmailVerification)

	if err := consumer.Handle(context.Background(), publishedBody(event)); err != nil {
		t.Fatalf("Handle: %v", err)
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
		t.Errorf("token query = %q, want the plaintext token the link is built from",
			verificationLink.Query().Get("token"))
	}
}

func TestDeliveringAMalformedBodyReportsTheEventItCouldNotRead(t *testing.T) {
	silenceDefaultLogger(t)

	gateway := email.NewInMemoryGateway()
	consumer := onlyConsumer(t, newTestRegistry(gateway))

	err := consumer.Handle(context.Background(), []byte(`{"challengeId": `))

	undecodable := ErrEventPayloadUndecodable{}
	if !errors.As(err, &undecodable) {
		t.Fatalf("error = %v (%T), want %T", err, err, undecodable)
	}
	if undecodable.Label != events.AuthChallengeGeneratedLabel {
		t.Errorf("Label = %q, want %q", undecodable.Label, events.AuthChallengeGeneratedLabel)
	}
	if undecodable.Cause == nil {
		t.Error("Cause = nil, want the json failure")
	}
	if len(gateway.Sent) != 0 {
		t.Errorf("sent messages = %d, want none for a body that never decoded", len(gateway.Sent))
	}
}

func TestNewConsumerWithoutAHandlerDeclaresNothingToDeliverTo(t *testing.T) {
	consumer := NewConsumer(NewConsumerParams[auth.AuthChallengeGeneratedData]{
		Label: events.AuthChallengeGeneratedLabel,
		Queue: AuthChallengeEmailQueue,
	})

	if consumer.Handle != nil {
		t.Error("Handle = non-nil, want none when there is no handler to call")
	}
	if consumer.Label != events.AuthChallengeGeneratedLabel || consumer.Queue != AuthChallengeEmailQueue {
		t.Errorf("consumer = %q on %q, want the declaration kept for the skip to report",
			consumer.Label, consumer.Queue)
	}
}

func TestEventPayloadUndecodableUnwrapsToItsCause(t *testing.T) {
	cause := errors.New("unexpected end of JSON input")
	err := ErrEventPayloadUndecodable{Label: events.AuthChallengeGeneratedLabel, Cause: cause}

	if !errors.Is(err, cause) {
		t.Errorf("errors.Is did not recover %v from %v", cause, err)
	}
}
