package messaging

import (
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/eventstest"
)

func TestMessageEventsCarryThePayloadSubscribersRead(t *testing.T) {
	messageID := uuid.MustParse("6f1c2d3e-4b5a-4c6d-8e7f-9a0b1c2d3e4f")
	recipientID := uuid.MustParse("1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15")
	message := func(extra map[string]any) map[string]any {
		payload := map[string]any{
			"messageId":      messageID.String(),
			"recipientId":    recipientID.String(),
			"recipientEmail": "accountant@example.com",
			"type":           "email_verification",
			"method":         "email",
		}
		for key, value := range extra {
			payload[key] = value
		}
		return payload
	}

	cases := []struct {
		name  string
		build func() (events.NewEvent, error)
		want  eventstest.WantEvent
	}{
		{
			name: "sent",
			build: func() (events.NewEvent, error) {
				return NewMessageSent(NewMessageSentParams{
					MessageID: messageID, RecipientID: recipientID, RecipientEmail: "accountant@example.com",
					Type: "email_verification", Method: "email", ReferenceID: "brevo-42",
				})
			},
			want: eventstest.WantEvent{
				Label:   events.MessageSentLabel,
				UserID:  recipientID,
				Payload: message(map[string]any{"referenceId": "brevo-42"}),
			},
		},
		{
			name: "failed",
			build: func() (events.NewEvent, error) {
				return NewMessageFailed(NewMessageFailedParams{
					MessageID: messageID, RecipientID: recipientID, RecipientEmail: "accountant@example.com",
					Type: "email_verification", Method: "email", Errors: []string{"brevo refused it", "rate limited"},
				})
			},
			want: eventstest.WantEvent{
				Label:   events.MessageFailedLabel,
				UserID:  recipientID,
				Payload: message(map[string]any{"errors": []any{"brevo refused it", "rate limited"}}),
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			built, err := testCase.build()
			if err != nil {
				t.Fatalf("building the event: %v", err)
			}

			eventstest.AssertEvent(t, built, testCase.want)
		})
	}
}
