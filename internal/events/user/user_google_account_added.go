package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

type userGoogleAccountAddedData struct {
	UserID uuid.UUID `json:"userId"`
}

// NewUserGoogleAccountAddedParams is the account a user.google_account_added event names; no google subject rides along.
type NewUserGoogleAccountAddedParams struct {
	UserID uuid.UUID
}

// NewUserGoogleAccountAdded builds the event announcing a password account connected a google account.
func NewUserGoogleAccountAdded(params NewUserGoogleAccountAddedParams) (events.NewEvent, error) {
	return events.New(events.NewEventParams{
		Label:  events.UserGoogleAccountAddedLabel,
		UserID: uuid.NullUUID{UUID: params.UserID, Valid: true},
		Data:   userGoogleAccountAddedData{UserID: params.UserID},
	})
}

// DispatchUserGoogleAccountAdded builds, records and publishes the event announcing a google account addition.
func DispatchUserGoogleAccountAdded(
	ctx context.Context,
	dispatcher events.Dispatcher,
	params NewUserGoogleAccountAddedParams,
) {
	dispatcher.Dispatch(ctx, func() (events.NewEvent, error) {
		return NewUserGoogleAccountAdded(params)
	})
}
