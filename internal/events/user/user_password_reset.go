package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

type userPasswordResetData struct {
	UserID uuid.UUID `json:"userId"`
}

// NewUserPasswordResetParams is the account a user.password_reset event names; no hashes ride along.
type NewUserPasswordResetParams struct {
	UserID uuid.UUID
}

// NewUserPasswordReset builds the event announcing an account set a new password through a reset link.
func NewUserPasswordReset(params NewUserPasswordResetParams) (events.NewEvent, error) {
	return events.New(events.NewEventParams{
		Label:  events.UserPasswordResetLabel,
		UserID: uuid.NullUUID{UUID: params.UserID, Valid: true},
		Data:   userPasswordResetData{UserID: params.UserID},
	})
}

// DispatchUserPasswordReset builds, records and publishes the event announcing a password reset.
func DispatchUserPasswordReset(
	ctx context.Context,
	dispatcher events.Dispatcher,
	params NewUserPasswordResetParams,
) {
	dispatcher.Dispatch(ctx, func() (events.NewEvent, error) {
		return NewUserPasswordReset(params)
	})
}
