package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/gateways/email"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/models"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

func (f sendVerificationEmailFixture) sendPasswordReset(params SendPasswordResetEmailParams) error {
	return SendPasswordResetEmail(context.Background(), SendPasswordResetEmailDependencies{
		Gateway:           f.gateway,
		MessageRepository: f.messageRepository,
		EventDispatcher:   f.captured.Dispatcher(),
		Logger:            slog.New(slog.NewTextHandler(f.logOutput, nil)),
		FrontendBaseURL:   localFrontendBaseURL,
	}, params)
}

func validPasswordResetParams(token string) SendPasswordResetEmailParams {
	return SendPasswordResetEmailParams{
		RecipientID:    recipientID,
		ChallengeID:    challengeID,
		RecipientEmail: recipientEmail,
		Token:          token,
	}
}

func onlySentPasswordResetEmail(t *testing.T, gateway *email.InMemoryGateway) email.PasswordResetEmailPayload {
	t.Helper()

	message, ok := gateway.OnlySentPasswordResetEmail()
	if !ok {
		t.Fatalf("sent password reset messages = %d, want 1", len(gateway.SentPasswordResets))
	}

	return message
}

func TestSendPasswordResetEmailHandsTheGatewayTheRecipientAndTheResetLink(t *testing.T) {
	fixture := newSendVerificationEmailFixture()

	if err := fixture.sendPasswordReset(validPasswordResetParams(plainToken)); err != nil {
		t.Fatalf("SendPasswordResetEmail: %v", err)
	}

	want := email.PasswordResetEmailPayload{
		RecipientEmail:         recipientEmail,
		PasswordResetLinkParam: "passwordResetLink",
		PasswordResetLink:      "http://localhost:3001/reset-password?token=abc123",
	}
	if message := onlySentPasswordResetEmail(t, fixture.gateway); message != want {
		t.Errorf("sent message = %#v, want %#v", message, want)
	}
	if len(fixture.gateway.Sent) != 0 {
		t.Errorf("verification messages = %d, want none", len(fixture.gateway.Sent))
	}
}

func TestSendPasswordResetEmailLogsAndAnnouncesAnAcceptedSend(t *testing.T) {
	fixture := newSendVerificationEmailFixture()

	if err := fixture.sendPasswordReset(validPasswordResetParams(plainToken)); err != nil {
		t.Fatalf("SendPasswordResetEmail: %v", err)
	}

	logged := onlyLoggedMessage(t, fixture.messageRepository)
	want := models.NewMessage{
		ID:             logged.ID,
		RecipientID:    recipientID,
		RecipientEmail: recipientEmail,
		Data:           json.RawMessage(`{"challengeId":"` + challengeID.String() + `"}`),
		Type:           models.PasswordReset,
		Method:         models.MethodEmail,
		Status:         models.StatusSent,
		ReferenceID:    email.InMemoryProviderMessageID(1),
		Errors:         []string{},
	}
	if !reflect.DeepEqual(logged, want) {
		t.Errorf("logged message = %#v, want %#v", logged, want)
	}
	if event := onlyDispatchedEvent(t, fixture.captured); event.Label != events.MessageSentLabel {
		t.Errorf("Label = %q, want %q", event.Label, events.MessageSentLabel)
	}
}

// The reset link hands over the account; the messaging log and its event must not keep the token.
func TestSendPasswordResetEmailKeepsTheTokenOutOfTheLogAndTheEvent(t *testing.T) {
	const distinctiveToken = "plaintext-secret-7f3a9c"

	fixture := newSendVerificationEmailFixture()

	if err := fixture.sendPasswordReset(validPasswordResetParams(distinctiveToken)); err != nil {
		t.Fatalf("SendPasswordResetEmail: %v", err)
	}

	logged := onlyLoggedMessage(t, fixture.messageRepository)
	if strings.Contains(fmt.Sprintf("%#v %s", logged, logged.Data), distinctiveToken) {
		t.Errorf("logged message = %#v, want no trace of the token", logged)
	}
	if event := onlyDispatchedEvent(t, fixture.captured); strings.Contains(string(event.Data), distinctiveToken) {
		t.Errorf("event payload = %s, want no trace of the token", event.Data)
	}
}

func TestSendPasswordResetEmailLogsAndAnnouncesARefusedSendAsFailed(t *testing.T) {
	providerFailure := errors.New("brevo refused it: status 400")

	fixture := newSendVerificationEmailFixture()
	fixture.gateway.SendFailure = providerFailure

	err := fixture.sendPasswordReset(validPasswordResetParams(plainToken))

	if _, sendFailed := errors.AsType[e.ErrSendPasswordResetEmailFailed](err); !sendFailed {
		t.Fatalf("error = %v (%T), want ErrSendPasswordResetEmailFailed", err, err)
	}
	if !errors.Is(err, providerFailure) {
		t.Errorf("error = %v, want it to unwrap to the gateway's failure", err)
	}

	logged := onlyLoggedMessage(t, fixture.messageRepository)
	if logged.Status != models.StatusFailed || logged.Type != models.PasswordReset {
		t.Errorf("logged = %q %q, want a failed password reset", logged.Status, logged.Type)
	}
	if event := onlyDispatchedEvent(t, fixture.captured); event.Label != events.MessageFailedLabel {
		t.Errorf("Label = %q, want %q", event.Label, events.MessageFailedLabel)
	}
}

func TestSendPasswordResetEmailRejectsMissingParamsBeforeSendingAnything(t *testing.T) {
	cases := []struct {
		name    string
		params  SendPasswordResetEmailParams
		wantErr func(err error) bool
	}{
		{
			name:   "no recipient id",
			params: SendPasswordResetEmailParams{RecipientEmail: recipientEmail, Token: plainToken},
			wantErr: func(err error) bool {
				_, matched := errors.AsType[e.ErrRecipientIDMissing](err)
				return matched
			},
		},
		{
			name:   "no recipient",
			params: SendPasswordResetEmailParams{RecipientID: recipientID, Token: plainToken},
			wantErr: func(err error) bool {
				_, matched := errors.AsType[e.ErrRecipientEmailMissing](err)
				return matched
			},
		},
		{
			name:   "no token",
			params: SendPasswordResetEmailParams{RecipientID: recipientID, RecipientEmail: recipientEmail},
			wantErr: func(err error) bool {
				_, matched := errors.AsType[e.ErrVerificationTokenMissing](err)
				return matched
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newSendVerificationEmailFixture()

			err := fixture.sendPasswordReset(testCase.params)

			if !testCase.wantErr(err) {
				t.Fatalf("error = %v (%T), want the missing param reported", err, err)
			}
			if len(fixture.gateway.SentPasswordResets) != 0 || len(fixture.messageRepository.Saved) != 0 {
				t.Error("sent or logged a message, want nothing for an incomplete request")
			}
		})
	}
}
