package messaging

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/gateways/email"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/usecases"
	"github.com/pauliusgin/web_app_starter-be/internal/events/auth"
)

func newTestPasswordResetEmailHandler(
	gateway *email.InMemoryGateway,
	messageRepository *repositories.InMemoryMessageWriteRepository,
) func(context.Context, auth.AuthChallengeGeneratedData) error {
	dependencies := usecases.SendPasswordResetEmailDependencies{
		Gateway:           gateway,
		MessageRepository: messageRepository,
		FrontendBaseURL:   testFrontendBaseURL,
	}

	return PasswordResetEmailOnChallenge(PasswordResetEmailDependencies{
		SendPasswordResetEmail: func(ctx context.Context, params usecases.SendPasswordResetEmailParams) error {
			return usecases.SendPasswordResetEmail(ctx, dependencies, params)
		},
	})
}

func TestPasswordResetEmailHandlerSendsTheResetLinkToTheChallengeTarget(t *testing.T) {
	silenceDefaultLogger(t)

	gateway := email.NewInMemoryGateway()
	messageRepository := repositories.NewInMemoryMessageWriteRepository()
	sendPasswordResetEmail := newTestPasswordResetEmailHandler(gateway, messageRepository)

	err := sendPasswordResetEmail(context.Background(), testAuthChallengePayload(challenge.PurposePasswordReset))
	if err != nil {
		t.Fatalf("sendPasswordResetEmail: %v", err)
	}

	message, ok := gateway.OnlySentPasswordResetEmail()
	if !ok {
		t.Fatalf("sent password reset messages = %d, want 1", len(gateway.SentPasswordResets))
	}
	if message.RecipientEmail != testChallengeTarget {
		t.Errorf("RecipientEmail = %q, want %q", message.RecipientEmail, testChallengeTarget)
	}

	resetLink, err := url.Parse(message.PasswordResetLink)
	if err != nil {
		t.Fatalf("parsing the built url %q: %v", message.PasswordResetLink, err)
	}
	if resetLink.Path != "/web_app_starter/reset-password" || resetLink.Query().Get("token") != testVerificationToken {
		t.Errorf("PasswordResetLink = %q, want the reset page carrying the challenge token", message.PasswordResetLink)
	}

	loggedMessage, ok := messageRepository.OnlySavedMessage()
	if !ok || loggedMessage.RecipientID != testUserID {
		t.Errorf("logged message = %+v, want one for the challenge's user %s", loggedMessage, testUserID)
	}
}

// Every auth challenge reaches this queue; a verification token must never go out behind a reset URL.
func TestPasswordResetEmailHandlerIgnoresOtherChallenges(t *testing.T) {
	silenceDefaultLogger(t)

	for _, purpose := range []challenge.Purpose{challenge.PurposeEmailVerification, challenge.PurposePhoneVerification} {
		t.Run(string(purpose), func(t *testing.T) {
			gateway := email.NewInMemoryGateway()
			messageRepository := repositories.NewInMemoryMessageWriteRepository()
			sendPasswordResetEmail := newTestPasswordResetEmailHandler(gateway, messageRepository)

			if err := sendPasswordResetEmail(context.Background(), testAuthChallengePayload(purpose)); err != nil {
				t.Fatalf("sendPasswordResetEmail = %v, want an ignored challenge to be no error", err)
			}
			if len(gateway.SentPasswordResets) != 0 || len(gateway.Sent) != 0 {
				t.Errorf("sent messages, want none for %q", purpose)
			}
			if len(messageRepository.Saved) != 0 {
				t.Errorf("saved messages = %d, want none logged for %q", len(messageRepository.Saved), purpose)
			}
		})
	}
}

func TestPasswordResetEmailHandlerReportsAFailingGateway(t *testing.T) {
	silenceDefaultLogger(t)

	providerFailure := errors.New("brevo refused the password reset email: status 400")
	gateway := email.NewInMemoryGateway()
	gateway.SendFailure = providerFailure

	sendPasswordResetEmail := newTestPasswordResetEmailHandler(gateway, repositories.NewInMemoryMessageWriteRepository())

	err := sendPasswordResetEmail(context.Background(), testAuthChallengePayload(challenge.PurposePasswordReset))

	if !errors.Is(err, providerFailure) {
		t.Errorf("error = %v, want it to unwrap to the gateway's own failure", err)
	}
}

func TestPasswordResetEmailHandlerIsNilWithoutASender(t *testing.T) {
	if PasswordResetEmailOnChallenge(PasswordResetEmailDependencies{}) != nil {
		t.Error("handler = non-nil, want none without a use case to call")
	}
}
