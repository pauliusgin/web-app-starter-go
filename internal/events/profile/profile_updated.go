package profile

import (
	"context"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

// NewProfileUpdatedParams is the user and the new names a profile.updated event announces.
type NewProfileUpdatedParams struct {
	UserID    uuid.UUID
	FirstName string
	LastName  string
}

// NewProfileUpdated builds the event announcing a user replaced their profile's names.
func NewProfileUpdated(params NewProfileUpdatedParams) (events.NewEvent, error) {
	return events.New(events.NewEventParams{
		Label:  events.ProfileUpdatedLabel,
		UserID: uuid.NullUUID{UUID: params.UserID, Valid: true},
		Data: profileEventData{
			UserID:    params.UserID,
			FirstName: params.FirstName,
			LastName:  params.LastName,
		},
		RedactedData: profileEventRedactedData{UserID: params.UserID},
	})
}

// DispatchProfileUpdated builds, records and publishes the event announcing an updated profile.
func DispatchProfileUpdated(ctx context.Context, dispatcher events.Dispatcher, params NewProfileUpdatedParams) {
	dispatcher.Dispatch(ctx, func() (events.NewEvent, error) {
		return NewProfileUpdated(params)
	})
}
