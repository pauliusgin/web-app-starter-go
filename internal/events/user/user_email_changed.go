package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

type userEmailChangedData struct {
	UserID        uuid.UUID `json:"userId"`
	PreviousEmail string    `json:"previousEmail"`
	Email         string    `json:"email"`
}

// NewUserEmailChangedParams is the account and the old and new addresses a user.email_changed carries.
type NewUserEmailChangedParams struct {
	UserID        uuid.UUID
	PreviousEmail string
	Email         string
}

// NewUserEmailChanged builds the event announcing an account switched to a newly verified email.
func NewUserEmailChanged(params NewUserEmailChangedParams) (events.NewEvent, error) {
	return events.New(events.NewEventParams{
		Label:  events.UserEmailChangedLabel,
		UserID: uuid.NullUUID{UUID: params.UserID, Valid: true},
		Data: userEmailChangedData{
			UserID:        params.UserID,
			PreviousEmail: params.PreviousEmail,
			Email:         params.Email,
		},
	})
}

// DispatchUserEmailChanged builds, records and publishes the event announcing a changed email.
func DispatchUserEmailChanged(ctx context.Context, dispatcher events.Dispatcher, params NewUserEmailChangedParams) {
	dispatcher.Dispatch(ctx, func() (events.NewEvent, error) {
		return NewUserEmailChanged(params)
	})
}
