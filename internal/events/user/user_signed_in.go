package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

type userSignedInData struct {
	UserID uuid.UUID `json:"userId"`
	Email  string    `json:"email"`
}

// NewUserSignedInParams is the account identity a user.signed_in event carries.
type NewUserSignedInParams struct {
	UserID uuid.UUID
	Email  string
}

// NewUserSignedIn builds the event announcing a successful sign-in.
func NewUserSignedIn(params NewUserSignedInParams) (events.NewEvent, error) {
	return events.New(events.NewEventParams{
		Label:  events.UserSignedInLabel,
		UserID: uuid.NullUUID{UUID: params.UserID, Valid: true},
		Data: userSignedInData{
			UserID: params.UserID,
			Email:  params.Email,
		},
	})
}

// DispatchUserSignedIn builds, records and publishes the event announcing a successful sign-in.
func DispatchUserSignedIn(ctx context.Context, dispatcher events.Dispatcher, params NewUserSignedInParams) {
	dispatcher.Dispatch(ctx, func() (events.NewEvent, error) {
		return NewUserSignedIn(params)
	})
}
