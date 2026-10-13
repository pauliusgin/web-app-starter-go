package usecases

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/gateways/email"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

const recipientEmail = "someone@example.com"
const localFrontendBaseURL = "http://localhost:3001"
const plainToken = "abc123"

var recipientID = uuid.MustParse("1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15")

var challengeID = uuid.MustParse("0b3e2c1d-7f6a-4e5b-8c9d-1a2b3c4d5e6f")

type capturedEvents struct {
	recorded  []events.NewEvent
	published []events.NewEvent
}

func (c *capturedEvents) dispatcher() events.Dispatcher {
	return events.Dispatcher{
		Recorder: func(_ context.Context, event events.NewEvent) error {
			c.recorded = append(c.recorded, event)
			return nil
		},
		Publisher: func(_ context.Context, event events.NewEvent) error {
			c.published = append(c.published, event)
			return nil
		},
	}
}

// sendVerificationEmailFixture wires the use case to in-memory collaborators it can be inspected through.
type sendVerificationEmailFixture struct {
	gateway           *email.InMemoryGateway
	messageRepository *repositories.InMemoryMessageWriteRepository
	captured          *capturedEvents
	logOutput         *bytes.Buffer
}

func newSendVerificationEmailFixture() sendVerificationEmailFixture {
	return sendVerificationEmailFixture{
		gateway:           email.NewInMemoryGateway(),
		messageRepository: repositories.NewInMemoryMessageWriteRepository(),
		captured:          &capturedEvents{},
		logOutput:         &bytes.Buffer{},
	}
}

func (f sendVerificationEmailFixture) dependencies() SendVerificationEmailDependencies {
	return SendVerificationEmailDependencies{
		Gateway:           f.gateway,
		MessageRepository: f.messageRepository,
		EventDispatcher:   f.captured.dispatcher(),
		Logger:            slog.New(slog.NewTextHandler(f.logOutput, nil)),
		FrontendBaseURL:   localFrontendBaseURL,
	}
}

func (f sendVerificationEmailFixture) send(params SendVerificationEmailParams) error {
	return SendVerificationEmail(context.Background(), f.dependencies(), params)
}

func validParams(token string) SendVerificationEmailParams {
	return SendVerificationEmailParams{
		RecipientID:    recipientID,
		ChallengeID:    challengeID,
		RecipientEmail: recipientEmail,
		Token:          token,
		Type:           models.EmailVerification,
	}
}

// onlySentMessage fails the test unless the gateway was asked to send exactly one message.
func onlySentMessage(t *testing.T, gateway *email.InMemoryGateway) email.VerificationEmailPayload {
	t.Helper()

	message, ok := gateway.OnlySentMessage()
	if !ok {
		t.Fatalf("sent messages = %d, want 1", len(gateway.Sent))
	}

	return message
}

// onlyLoggedMessage fails the test unless exactly one message reached the messaging log.
func onlyLoggedMessage(t *testing.T, repository *repositories.InMemoryMessageWriteRepository) models.NewMessage {
	t.Helper()

	message, ok := repository.OnlySavedMessage()
	if !ok {
		t.Fatalf("logged messages = %d, want 1", len(repository.Saved))
	}

	return message
}

// onlyDispatchedEvent fails the test unless exactly one event was both recorded and published.
func onlyDispatchedEvent(t *testing.T, captured *capturedEvents) events.NewEvent {
	t.Helper()

	if len(captured.recorded) != 1 || len(captured.published) != 1 {
		t.Fatalf("recorded = %d, published = %d, want 1 each", len(captured.recorded), len(captured.published))
	}

	return captured.published[0]
}

func decodeEventPayload(t *testing.T, event events.NewEvent) map[string]any {
	t.Helper()

	payload := map[string]any{}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		t.Fatalf("event payload %s is not valid JSON: %v", event.Data, err)
	}

	return payload
}

func TestSendVerificationEmailHandsTheGatewayTheRecipientAndTheBuiltLink(t *testing.T) {
	fixture := newSendVerificationEmailFixture()

	if err := fixture.send(validParams(plainToken)); err != nil {
		t.Fatalf("SendVerificationEmail: %v", err)
	}

	want := email.VerificationEmailPayload{
		RecipientEmail:        recipientEmail,
		VerificationLinkParam: verificationLinkParam,
		VerificationLink:      "http://localhost:3001/verify-email?token=abc123",
	}
	if message := onlySentMessage(t, fixture.gateway); message != want {
		t.Errorf("sent message = %#v, want %#v", message, want)
	}
}

func TestSendVerificationEmailRejectsMissingParamsBeforeSendingAnything(t *testing.T) {
	cases := []struct {
		name        string
		params      SendVerificationEmailParams
		wantErrType any
	}{
		{
			name:        "no recipient id",
			params:      SendVerificationEmailParams{RecipientEmail: recipientEmail, Token: plainToken},
			wantErrType: &e.ErrRecipientIDMissing{},
		},
		{
			name:        "no recipient",
			params:      SendVerificationEmailParams{RecipientID: recipientID, Token: plainToken},
			wantErrType: &e.ErrRecipientEmailMissing{},
		},
		{
			name:        "no token",
			params:      SendVerificationEmailParams{RecipientID: recipientID, RecipientEmail: recipientEmail},
			wantErrType: &e.ErrVerificationTokenMissing{},
		},
		{
			name:        "nothing at all reports the recipient id first",
			params:      SendVerificationEmailParams{},
			wantErrType: &e.ErrRecipientIDMissing{},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newSendVerificationEmailFixture()

			err := fixture.send(testCase.params)

			if !errors.As(err, testCase.wantErrType) {
				t.Fatalf("error = %v (%T), want %T", err, err, testCase.wantErrType)
			}
			if len(fixture.gateway.Sent) != 0 {
				t.Errorf("sent messages = %d, want none for an incomplete request", len(fixture.gateway.Sent))
			}
			if len(fixture.messageRepository.Saved) != 0 {
				t.Errorf("logged messages = %d, want none for a message never sent",
					len(fixture.messageRepository.Saved))
			}
			if len(fixture.captured.published) != 0 {
				t.Errorf("published events = %d, want none for a message never sent",
					len(fixture.captured.published))
			}
		})
	}
}

func TestSendVerificationEmailBuildsTheVerificationLink(t *testing.T) {
	cases := []struct {
		name        string
		token       string
		messageType models.MessageType
		wantLink    string
	}{
		{
			name:        "email verification",
			token:       plainToken,
			messageType: models.EmailVerification,
			wantLink:    "http://localhost:3001/verify-email?token=abc123",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			fixture := newSendVerificationEmailFixture()
			params := validParams(testCase.token)
			params.Type = testCase.messageType

			if err := fixture.send(params); err != nil {
				t.Fatalf("SendVerificationEmail: %v", err)
			}

			verificationLink := onlySentMessage(t, fixture.gateway).VerificationLink
			if verificationLink != testCase.wantLink {
				t.Errorf("VerificationLink = %q, want %q", verificationLink, testCase.wantLink)
			}

			parsedLink, err := url.Parse(verificationLink)
			if err != nil {
				t.Fatalf("parse VerificationLink %q: %v", verificationLink, err)
			}
			if token := parsedLink.Query().Get(VerificationTokenQueryParam); token != testCase.token {
				t.Errorf("token read back from link = %q, want %q", token, testCase.token)
			}
		})
	}
}

func TestSendVerificationEmailLogsAnAcceptedSendAsSent(t *testing.T) {
	fixture := newSendVerificationEmailFixture()

	if err := fixture.send(validParams(plainToken)); err != nil {
		t.Fatalf("SendVerificationEmail: %v", err)
	}

	logged := onlyLoggedMessage(t, fixture.messageRepository)

	want := models.NewMessage{
		ID:             logged.ID,
		RecipientID:    recipientID,
		RecipientEmail: recipientEmail,
		Data:           json.RawMessage(`{"challengeId":"` + challengeID.String() + `"}`),
		Type:           models.EmailVerification,
		Method:         models.MethodEmail,
		Status:         models.StatusSent,
		ReferenceID:    email.InMemoryProviderMessageID(1),
		Errors:         []string{},
	}
	if !reflect.DeepEqual(logged, want) {
		t.Errorf("logged message = %#v, want %#v", logged, want)
	}
}

// The verification link carries the plaintext token; the messaging log is not a place to keep it.
func TestSendVerificationEmailKeepsTheTokenOutOfTheLogAndTheEvent(t *testing.T) {
	const distinctiveToken = "plaintext-secret-7f3a9c"

	fixture := newSendVerificationEmailFixture()

	if err := fixture.send(validParams(distinctiveToken)); err != nil {
		t.Fatalf("SendVerificationEmail: %v", err)
	}

	logged := onlyLoggedMessage(t, fixture.messageRepository)
	if strings.Contains(fmt.Sprintf("%#v %s", logged, logged.Data), distinctiveToken) {
		t.Errorf("logged message = %#v, want no trace of the token", logged)
	}

	event := onlyDispatchedEvent(t, fixture.captured)
	if strings.Contains(string(event.Data), distinctiveToken) {
		t.Errorf("event payload = %s, want no trace of the token", event.Data)
	}
}

func TestSendVerificationEmailAnnouncesAnAcceptedSend(t *testing.T) {
	fixture := newSendVerificationEmailFixture()

	if err := fixture.send(validParams(plainToken)); err != nil {
		t.Fatalf("SendVerificationEmail: %v", err)
	}

	logged := onlyLoggedMessage(t, fixture.messageRepository)
	event := onlyDispatchedEvent(t, fixture.captured)

	if event.Label != events.MessageSentLabel {
		t.Errorf("Label = %q, want %q", event.Label, events.MessageSentLabel)
	}

	payload := decodeEventPayload(t, event)
	if payload["messageId"] != logged.ID.String() {
		t.Errorf("messageId = %v, want the logged message's id %s", payload["messageId"], logged.ID)
	}
	if payload["referenceId"] != email.InMemoryProviderMessageID(1) {
		t.Errorf("referenceId = %v, want the provider's id", payload["referenceId"])
	}
}

func TestSendVerificationEmailLogsAndAnnouncesARefusedSendAsFailed(t *testing.T) {
	providerFailure := errors.New("brevo refused it: status 400")

	fixture := newSendVerificationEmailFixture()
	fixture.gateway.SendFailure = providerFailure

	err := fixture.send(validParams(plainToken))

	sendFailed := e.ErrSendVerificationEmailFailed{}
	if !errors.As(err, &sendFailed) {
		t.Fatalf("error = %v (%T), want %T", err, err, sendFailed)
	}
	if !errors.Is(err, providerFailure) {
		t.Errorf("error = %v, want it to unwrap to the gateway's failure", err)
	}

	logged := onlyLoggedMessage(t, fixture.messageRepository)
	if logged.Status != models.StatusFailed {
		t.Errorf("Status = %q, want %q", logged.Status, models.StatusFailed)
	}
	if !reflect.DeepEqual(logged.Errors, []string{providerFailure.Error()}) {
		t.Errorf("Errors = %#v, want the gateway's failure", logged.Errors)
	}
	if logged.ReferenceID != "" {
		t.Errorf("ReferenceID = %q, want none for a refused send", logged.ReferenceID)
	}

	event := onlyDispatchedEvent(t, fixture.captured)
	if event.Label != events.MessageFailedLabel {
		t.Errorf("Label = %q, want %q", event.Label, events.MessageFailedLabel)
	}

	payload := decodeEventPayload(t, event)
	if !reflect.DeepEqual(payload["errors"], []any{providerFailure.Error()}) {
		t.Errorf("errors = %#v, want the gateway's failure", payload["errors"])
	}
}

func TestSendVerificationEmailLogsEachJoinedGatewayError(t *testing.T) {
	fixture := newSendVerificationEmailFixture()
	fixture.gateway.SendFailure = errors.Join(errors.New("rate limited"), errors.New("retry later"))

	if err := fixture.send(validParams(plainToken)); err == nil {
		t.Fatal("SendVerificationEmail = nil, want the gateway's failure reported")
	}

	logged := onlyLoggedMessage(t, fixture.messageRepository)
	if !reflect.DeepEqual(logged.Errors, []string{"rate limited", "retry later"}) {
		t.Errorf("Errors = %#v, want one entry per joined error", logged.Errors)
	}
}

// By the time the log is written the email has left; failing here would only get it redelivered
// and sent twice.
func TestSendVerificationEmailSucceedsWhenTheMessagingLogFails(t *testing.T) {
	fixture := newSendVerificationEmailFixture()
	fixture.messageRepository.SaveFailure = errors.New("database unreachable")

	if err := fixture.send(validParams(plainToken)); err != nil {
		t.Fatalf("SendVerificationEmail = %v, want a failed log not to fail the send", err)
	}

	onlySentMessage(t, fixture.gateway)
	onlyDispatchedEvent(t, fixture.captured)

	if !strings.Contains(fixture.logOutput.String(), "database unreachable") {
		t.Errorf("log output = %q, want the save failure logged", fixture.logOutput.String())
	}
}

func TestSendVerificationEmailLogsTheMessageTypeItWasAskedFor(t *testing.T) {
	fixture := newSendVerificationEmailFixture()
	params := validParams(plainToken)
	params.Type = models.EmailVerification

	if err := fixture.send(params); err != nil {
		t.Fatalf("SendVerificationEmail: %v", err)
	}

	loggedMessage := onlyLoggedMessage(t, fixture.messageRepository)
	if loggedMessage.Type != models.EmailVerification {
		t.Errorf("logged Type = %q, want %q", loggedMessage.Type, models.EmailVerification)
	}
}

func TestSendVerificationEmailRejectsATypeWithoutALinkBeforeSendingAnything(t *testing.T) {
	fixture := newSendVerificationEmailFixture()
	params := validParams(plainToken)
	params.Type = ""

	err := fixture.send(params)

	unsupported := e.ErrVerificationMessageTypeUnsupported{}
	if !errors.As(err, &unsupported) {
		t.Fatalf("error = %v, want ErrVerificationMessageTypeUnsupported", err)
	}
	if len(fixture.gateway.Sent) != 0 {
		t.Errorf("sent messages = %d, want none for an unsupported type", len(fixture.gateway.Sent))
	}
	if len(fixture.messageRepository.Saved) != 0 {
		t.Errorf("logged messages = %d, want none for a message never sent", len(fixture.messageRepository.Saved))
	}
}
