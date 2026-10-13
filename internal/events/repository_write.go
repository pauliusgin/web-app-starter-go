package events

import "context"

// WriteRepository is the event_log write port the domain depends on.
type WriteRepository interface {
	Save(ctx context.Context, newEvent NewEvent) (Event, error)
}
