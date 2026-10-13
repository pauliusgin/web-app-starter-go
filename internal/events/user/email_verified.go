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

// EmailVerifiedParams is the account identity a user.signed_up event carries.
type EmailVerifiedParams struct {
	UserID uuid.UUID
	Email  string
}

// EmailVerified builds the event announcing a successful email verification.
func EmailVerified(params EmailVerifiedParams) (events.NewEvent, error) {
	return events.New(events.NewEventParams{
		Label:  events.EmailVerifiedLabel,
		UserID: uuid.NullUUID{UUID: params.UserID, Valid: true},
		Data: emailVerifiedData{
			UserID: params.UserID,
			Email:  params.Email,
		},
	})
}

// DispatchEmailVerified builds, records and publishes the event announcing a verified email.
func DispatchEmailVerified(ctx context.Context, dispatcher events.Dispatcher, params EmailVerifiedParams) {
	dispatcher.Dispatch(ctx, func() (events.NewEvent, error) {
		return EmailVerified(params)
	})
}
