package upload

import (
	"context"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

type uploadDeletedData struct {
	UploadID uuid.UUID `json:"uploadId"`
	UserID   uuid.UUID `json:"userId"`
	Purpose  string    `json:"purpose"`
}

// NewUploadDeletedParams is the upload an upload.deleted event names.
type NewUploadDeletedParams struct {
	UploadID uuid.UUID
	UserID   uuid.UUID
	Purpose  string
}

// NewUploadDeleted builds the event announcing an upload and its stored files were removed.
func NewUploadDeleted(params NewUploadDeletedParams) (events.NewEvent, error) {
	return events.New(events.NewEventParams{
		Label:  events.UploadDeletedLabel,
		UserID: uuid.NullUUID{UUID: params.UserID, Valid: true},
		Data: uploadDeletedData{
			UploadID: params.UploadID,
			UserID:   params.UserID,
			Purpose:  params.Purpose,
		},
	})
}

// DispatchUploadDeleted builds, records and publishes the event announcing a removed upload.
func DispatchUploadDeleted(ctx context.Context, dispatcher events.Dispatcher, params NewUploadDeletedParams) {
	dispatcher.Dispatch(ctx, func() (events.NewEvent, error) {
		return NewUploadDeleted(params)
	})
}
