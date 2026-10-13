// Package messaging defines the events dispatched about messages sent to users.
package messaging

import (
	"context"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

type messageSentData struct {
	MessageID      uuid.UUID `json:"messageId"`
	RecipientID    uuid.UUID `json:"recipientId"`
	RecipientEmail string    `json:"recipientEmail"`
	Type           string    `json:"type"`
	Method         string    `json:"method"`
	ReferenceID    string    `json:"referenceId"`
}

// MessageSentParams is the logged message a message.sent event carries.
type MessageSentParams struct {
	MessageID      uuid.UUID
	RecipientID    uuid.UUID
	RecipientEmail string
	Type           string
	Method         string
	ReferenceID    string
}

// MessageSent builds the event announcing a message the delivery provider accepted.
func MessageSent(params MessageSentParams) (events.NewEvent, error) {
	return events.New(events.NewEventParams{
		Label:  events.MessageSentLabel,
		UserID: uuid.NullUUID{UUID: params.RecipientID, Valid: true},
		Data: messageSentData{
			MessageID:      params.MessageID,
			RecipientID:    params.RecipientID,
			RecipientEmail: params.RecipientEmail,
			Type:           params.Type,
			Method:         params.Method,
			ReferenceID:    params.ReferenceID,
		},
	})
}

// DispatchMessageSent builds, records and publishes the event announcing an accepted message.
func DispatchMessageSent(ctx context.Context, dispatcher events.Dispatcher, params MessageSentParams) {
	dispatcher.Dispatch(ctx, func() (events.NewEvent, error) {
		return MessageSent(params)
	})
}
