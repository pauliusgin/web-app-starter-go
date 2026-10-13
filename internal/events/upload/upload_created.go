// Package upload builds and dispatches the events announcing changes to uploaded files.
package upload

import (
	"context"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

type uploadCreatedData struct {
	UploadID uuid.UUID `json:"uploadId"`
	UserID   uuid.UUID `json:"userId"`
	Purpose  string    `json:"purpose"`
	Kind     string    `json:"kind"`
}

// NewUploadCreatedParams is the upload an upload.created event names.
type NewUploadCreatedParams struct {
	UploadID uuid.UUID
	UserID   uuid.UUID
	Purpose  string
	Kind     string
}

// NewUploadCreated builds the event announcing a user's file was stored and recorded as pending.
func NewUploadCreated(params NewUploadCreatedParams) (events.NewEvent, error) {
	return events.New(events.NewEventParams{
		Label:  events.UploadCreatedLabel,
		UserID: uuid.NullUUID{UUID: params.UserID, Valid: true},
		Data: uploadCreatedData{
			UploadID: params.UploadID,
			UserID:   params.UserID,
			Purpose:  params.Purpose,
			Kind:     params.Kind,
		},
	})
}

// DispatchUploadCreated builds, records and publishes the event announcing a stored upload.
func DispatchUploadCreated(ctx context.Context, dispatcher events.Dispatcher, params NewUploadCreatedParams) {
	dispatcher.Dispatch(ctx, func() (events.NewEvent, error) {
		return NewUploadCreated(params)
	})
}
