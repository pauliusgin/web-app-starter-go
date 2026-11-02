package events

import "context"

// WriteRepository is the event_log write port the domain depends on.
type WriteRepository interface {
	// SaveEvent records the event in the event log and returns it as stored.
	SaveEvent(ctx context.Context, newEvent NewEvent) (Event, error)
}
