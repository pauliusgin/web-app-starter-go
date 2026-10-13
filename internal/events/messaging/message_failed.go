package messaging

import (
	"context"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

type messageFailedData struct {
	MessageID      uuid.UUID `json:"messageId"`
	RecipientID    uuid.UUID `json:"recipientId"`
	RecipientEmail string    `json:"recipientEmail"`
	Type           string    `json:"type"`
	Method         string    `json:"method"`
	Errors         []string  `json:"errors"`
}

// MessageFailedParams is the logged message a message.failed event carries.
type MessageFailedParams struct {
	MessageID      uuid.UUID
	RecipientID    uuid.UUID
	RecipientEmail string
	Type           string
	Method         string
	Errors         []string
}

// MessageFailed builds the event announcing a message the delivery provider did not accept.
func MessageFailed(params MessageFailedParams) (events.NewEvent, error) {
	return events.New(events.NewEventParams{
		Label:  events.MessageFailedLabel,
		UserID: uuid.NullUUID{UUID: params.RecipientID, Valid: true},
		Data: messageFailedData{
			MessageID:      params.MessageID,
			RecipientID:    params.RecipientID,
			RecipientEmail: params.RecipientEmail,
			Type:           params.Type,
			Method:         params.Method,
			Errors:         params.Errors,
		},
	})
}

// DispatchMessageFailed builds, records and publishes the event announcing a failed message.
func DispatchMessageFailed(ctx context.Context, dispatcher events.Dispatcher, params MessageFailedParams) {
	dispatcher.Dispatch(ctx, func() (events.NewEvent, error) {
		return MessageFailed(params)
	})
}
