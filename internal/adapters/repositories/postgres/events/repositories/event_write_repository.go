// Package postgres adapts the generated sqlc queries to the event log write port.
package postgres

import (
	"context"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/events/mappers"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

var _ events.WriteRepository = EventWriteRepository{}

// EventWriteRepository stores domain events through the generated sqlc queries.
type EventWriteRepository struct {
	queries *sqlc.Queries
}

// NewEventWriteRepository builds a repository that writes events through the given sqlc queries.
func NewEventWriteRepository(queries *sqlc.Queries) EventWriteRepository {
	return EventWriteRepository{queries: queries}
}

// Save inserts the event and returns the row the database produced.
func (r EventWriteRepository) Save(ctx context.Context, newEvent events.NewEvent) (events.Event, error) {
	row, err := r.queries.RecordEvent(ctx, sqlc.RecordEventParams{
		ID:         newEvent.ID,
		UserID:     newEvent.UserID,
		EventLabel: string(newEvent.Label),
		EventData:  newEvent.Data,
	})
	if err != nil {
		return events.Event{}, events.ErrRecordEventFailed{Label: newEvent.Label, Cause: err}
	}

	return mappers.EventMapper(row), nil
}
