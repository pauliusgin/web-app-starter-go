package events

import (
	"context"
)

// Recorder records a dispatched event to the event log; the server wires one from Record.
type Recorder func(ctx context.Context, event NewEvent) error

// RecordDependencies carries the event log store the event is written to.
type RecordDependencies struct {
	EventWriteRepository WriteRepository
}

// Record writes one event to the event log and returns the row the database produced.
func Record(ctx context.Context, d RecordDependencies, e NewEvent) (Event, error) {
	return d.EventWriteRepository.SaveEvent(ctx, recordableEvent(e))
}

func recordableEvent(e NewEvent) NewEvent {
	if e.RedactedData == nil {
		return e
	}

	recordable := e
	recordable.Data = e.RedactedData
	recordable.RedactedData = nil

	return recordable
}
