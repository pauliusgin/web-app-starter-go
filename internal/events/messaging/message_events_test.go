package messaging

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

const testRecipientEmail = "accountant@example.com"
const testMessageType = "email_verification"
const testMethod = "email"

type assertMessageEventParams struct {
	event       events.NewEvent
	wantLabel   events.Label
	wantPayload map[string]any
	recipientID uuid.UUID
}

func assertMessageEvent(t *testing.T, params assertMessageEventParams) {
	t.Helper()

	if params.event.Label != params.wantLabel {
		t.Errorf("Label = %q, want %q", params.event.Label, params.wantLabel)
	}
	if params.event.UserID != (uuid.NullUUID{UUID: params.recipientID, Valid: true}) {
		t.Errorf("UserID = %+v, want the valid recipient %s", params.event.UserID, params.recipientID)
	}
	if params.event.RedactedData != nil {
		t.Errorf("RedactedData = %s, want nil for a payload without secrets", params.event.RedactedData)
	}

	payload := map[string]any{}
	if err := json.Unmarshal(params.event.Data, &payload); err != nil {
		t.Fatalf("the encoded payload %s is not valid JSON: %v", params.event.Data, err)
	}
	if !reflect.DeepEqual(payload, params.wantPayload) {
		t.Errorf("payload = %#v, want %#v", payload, params.wantPayload)
	}
}

func TestMessageSentCarriesTheMessageAndTheProviderReference(t *testing.T) {
	messageID := uuid.New()
	recipientID := uuid.New()

	sent, err := MessageSent(MessageSentParams{
		MessageID:      messageID,
		RecipientID:    recipientID,
		RecipientEmail: testRecipientEmail,
		Type:           testMessageType,
		Method:         testMethod,
		ReferenceID:    "brevo-42",
	})
	if err != nil {
		t.Fatalf("MessageSent: %v", err)
	}

	assertMessageEvent(t, assertMessageEventParams{
		event:       sent,
		wantLabel:   events.MessageSentLabel,
		recipientID: recipientID,
		wantPayload: map[string]any{
			"messageId":      messageID.String(),
			"recipientId":    recipientID.String(),
			"recipientEmail": testRecipientEmail,
			"type":           testMessageType,
			"method":         testMethod,
			"referenceId":    "brevo-42",
		},
	})
}

func TestMessageFailedCarriesTheMessageAndItsErrors(t *testing.T) {
	messageID := uuid.New()
	recipientID := uuid.New()

	failed, err := MessageFailed(MessageFailedParams{
		MessageID:      messageID,
		RecipientID:    recipientID,
		RecipientEmail: testRecipientEmail,
		Type:           testMessageType,
		Method:         testMethod,
		Errors:         []string{"brevo refused it", "rate limited"},
	})
	if err != nil {
		t.Fatalf("MessageFailed: %v", err)
	}

	assertMessageEvent(t, assertMessageEventParams{
		event:       failed,
		wantLabel:   events.MessageFailedLabel,
		recipientID: recipientID,
		wantPayload: map[string]any{
			"messageId":      messageID.String(),
			"recipientId":    recipientID.String(),
			"recipientEmail": testRecipientEmail,
			"type":           testMessageType,
			"method":         testMethod,
			"errors":         []any{"brevo refused it", "rate limited"},
		},
	})
}
