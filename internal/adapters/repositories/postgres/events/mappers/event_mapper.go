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

// RecordEventParamsMapper maps a new event onto the RecordEvent insert parameters.
func RecordEventParamsMapper(newEvent events.NewEvent) sqlc.RecordEventParams {
	return sqlc.RecordEventParams{
		ID:         newEvent.ID,
		UserID:     newEvent.UserID,
		EventLabel: string(newEvent.Label),
		EventData:  newEvent.Data,
	}
}
