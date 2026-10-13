// Package events defines the domain events the application dispatches, announces on the broker, and
// records in the event_log table.
package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Event is an event as stored in the event_log table.
type Event struct {
	ID         uuid.UUID
	UserID     uuid.NullUUID
	Label      Label
	Data       json.RawMessage
	OccurredAt time.Time
}

// NewEvent is an event that has been dispatched, before the database stamps OccurredAt.
type NewEvent struct {
	ID     uuid.UUID
	UserID uuid.NullUUID
	Label  Label

	// Data is the payload subscribers receive. RedactedData, when set, is the copy without secrets the
	// event log stores in its place: Data carries a secret the event log must not hold.
	Data         json.RawMessage
	RedactedData json.RawMessage
}

// NewEventParams is what an event constructor supplies; the id is generated, not passed.
type NewEventParams struct {
	Label        Label
	UserID       uuid.NullUUID
	Data         any
	RedactedData any
}

// New builds an event from its payloads; the per-event constructors in child packages call it.
func New(params NewEventParams) (NewEvent, error) {
	encodedData, err := json.Marshal(params.Data)
	if err != nil {
		return NewEvent{}, ErrEventDataUnencodable{Label: params.Label, Cause: err}
	}

	event := NewEvent{
		ID:     uuid.New(),
		UserID: params.UserID,
		Label:  params.Label,
		Data:   encodedData,
	}

	if params.RedactedData == nil {
		return event, nil
	}

	encodedRedactedData, err := json.Marshal(params.RedactedData)
	if err != nil {
		return NewEvent{}, ErrEventDataUnencodable{Label: params.Label, Cause: err}
	}

	event.RedactedData = encodedRedactedData

	return event, nil
}
