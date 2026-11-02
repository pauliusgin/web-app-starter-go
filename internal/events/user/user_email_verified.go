// Package user defines the events dispatched about an account's own lifecycle.
package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

type emailVerifiedData struct {
	UserID uuid.UUID `json:"userId"`
	Email  string    `json:"email"`
}

// NewUserEmailVerifiedParams is the account identity a user.email_verified event carries.
type NewUserEmailVerifiedParams struct {
	UserID uuid.UUID
	Email  string
}

// NewUserEmailVerified builds the event announcing a successful email verification.
func NewUserEmailVerified(params NewUserEmailVerifiedParams) (events.NewEvent, error) {
	return events.New(events.NewEventParams{
		Label:  events.UserEmailVerifiedLabel,
		UserID: uuid.NullUUID{UUID: params.UserID, Valid: true},
		Data: emailVerifiedData{
			UserID: params.UserID,
			Email:  params.Email,
		},
	})
}

// DispatchUserEmailVerified builds, records and publishes the event announcing a verified email.
func DispatchUserEmailVerified(ctx context.Context, dispatcher events.Dispatcher, params NewUserEmailVerifiedParams) {
	dispatcher.Dispatch(ctx, func() (events.NewEvent, error) {
		return NewUserEmailVerified(params)
	})
}
