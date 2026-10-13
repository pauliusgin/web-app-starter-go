// Package user defines the events dispatched about an account's own lifecycle.
package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

type userSignedUpData struct {
	UserID uuid.UUID `json:"userId"`
	Email  string    `json:"email"`
}

// NewUserSignedUpParams is the account identity a user.signed_up event carries.
type NewUserSignedUpParams struct {
	UserID uuid.UUID
	Email  string
}

// NewUserSignedUp builds the event announcing a completed signup.
func NewUserSignedUp(params NewUserSignedUpParams) (events.NewEvent, error) {
	return events.New(events.NewEventParams{
		Label:  events.UserSignedUpLabel,
		UserID: uuid.NullUUID{UUID: params.UserID, Valid: true},
		Data: userSignedUpData{
			UserID: params.UserID,
			Email:  params.Email,
		},
	})
}

// DispatchUserSignedUp builds, records and publishes the event announcing a completed signup.
func DispatchUserSignedUp(ctx context.Context, dispatcher events.Dispatcher, params NewUserSignedUpParams) {
	dispatcher.Dispatch(ctx, func() (events.NewEvent, error) {
		return NewUserSignedUp(params)
	})
}
