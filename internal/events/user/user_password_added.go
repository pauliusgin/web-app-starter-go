package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

type userPasswordAddedData struct {
	UserID uuid.UUID `json:"userId"`
}

// NewUserPasswordAddedParams is the account a user.password_added event names; no hashes ride along.
type NewUserPasswordAddedParams struct {
	UserID uuid.UUID
}

// NewUserPasswordAdded builds the event announcing a google-only account added a password.
func NewUserPasswordAdded(params NewUserPasswordAddedParams) (events.NewEvent, error) {
	return events.New(events.NewEventParams{
		Label:  events.UserPasswordAddedLabel,
		UserID: uuid.NullUUID{UUID: params.UserID, Valid: true},
		Data:   userPasswordAddedData{UserID: params.UserID},
	})
}

// DispatchUserPasswordAdded builds, records and publishes the event announcing a password addition.
func DispatchUserPasswordAdded(
	ctx context.Context,
	dispatcher events.Dispatcher,
	params NewUserPasswordAddedParams,
) {
	dispatcher.Dispatch(ctx, func() (events.NewEvent, error) {
		return NewUserPasswordAdded(params)
	})
}
