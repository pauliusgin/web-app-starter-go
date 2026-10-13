package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

type userDeletedData struct {
	UserID uuid.UUID `json:"userId"`
}

// NewUserDeletedParams is the account a user.deleted event names; the email is left out as PII.
type NewUserDeletedParams struct {
	UserID uuid.UUID
}

// NewUserDeleted builds the event announcing an account was soft-deleted.
func NewUserDeleted(params NewUserDeletedParams) (events.NewEvent, error) {
	return events.New(events.NewEventParams{
		Label:  events.UserDeletedLabel,
		UserID: uuid.NullUUID{UUID: params.UserID, Valid: true},
		Data:   userDeletedData{UserID: params.UserID},
	})
}

// DispatchUserDeleted builds, records and publishes the event announcing an account was soft-deleted.
func DispatchUserDeleted(ctx context.Context, dispatcher events.Dispatcher, params NewUserDeletedParams) {
	dispatcher.Dispatch(ctx, func() (events.NewEvent, error) {
		return NewUserDeleted(params)
	})
}
