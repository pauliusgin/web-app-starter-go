// Package mappers converts stored event_log rows into domain events.
package mappers

import (
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

// EventMapper maps a stored event_log row onto the domain type.
func EventMapper(row sqlc.EventLog) events.Event {
	mapped := events.Event{
		ID:         row.ID,
		UserID:     row.UserID,
		Label:      events.Label(row.EventLabel),
		Data:       row.EventData,
		OccurredAt: row.OccurredAt,
	}

	return mapped
}
