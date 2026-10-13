// Package profile builds and dispatches the events announcing changes to user profiles.
package profile

import (
	"context"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

type profileEventData struct {
	UserID    uuid.UUID `json:"userId"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
}

type profileEventRedactedData struct {
	UserID uuid.UUID `json:"userId"`
}

// NewProfileCreatedParams is the user and the names a profile.created event announces.
type NewProfileCreatedParams struct {
	UserID    uuid.UUID
	FirstName string
	LastName  string
}

// NewProfileCreated builds the event announcing a user recorded their profile.
func NewProfileCreated(params NewProfileCreatedParams) (events.NewEvent, error) {
	return events.New(events.NewEventParams{
		Label:  events.ProfileCreatedLabel,
		UserID: uuid.NullUUID{UUID: params.UserID, Valid: true},
		Data: profileEventData{
			UserID:    params.UserID,
			FirstName: params.FirstName,
			LastName:  params.LastName,
		},
		RedactedData: profileEventRedactedData{UserID: params.UserID},
	})
}

// DispatchProfileCreated builds, records and publishes the event announcing a recorded profile.
func DispatchProfileCreated(ctx context.Context, dispatcher events.Dispatcher, params NewProfileCreatedParams) {
	dispatcher.Dispatch(ctx, func() (events.NewEvent, error) {
		return NewProfileCreated(params)
	})
}
