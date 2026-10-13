package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

type userPasswordChangedData struct {
	UserID uuid.UUID `json:"userId"`
}

// NewUserPasswordChangedParams is the account a user.password_changed event names; no hashes ride along.
type NewUserPasswordChangedParams struct {
	UserID uuid.UUID
}

// NewUserPasswordChanged builds the event announcing an account replaced its password.
func NewUserPasswordChanged(params NewUserPasswordChangedParams) (events.NewEvent, error) {
	return events.New(events.NewEventParams{
		Label:  events.UserPasswordChangedLabel,
		UserID: uuid.NullUUID{UUID: params.UserID, Valid: true},
		Data:   userPasswordChangedData{UserID: params.UserID},
	})
}

// DispatchUserPasswordChanged builds, records and publishes the event announcing a password change.
func DispatchUserPasswordChanged(
	ctx context.Context,
	dispatcher events.Dispatcher,
	params NewUserPasswordChangedParams,
) {
	dispatcher.Dispatch(ctx, func() (events.NewEvent, error) {
		return NewUserPasswordChanged(params)
	})
}
