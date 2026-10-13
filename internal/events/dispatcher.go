package events

import (
	"context"
	"log/slog"
)

// Dispatcher records events in the event log and publishes them on the broker.
type Dispatcher struct {
	Recorder  Recorder
	Publisher Publisher
}

// EventBuilder constructs an event; the per-event constructors in child packages fit it.
type EventBuilder func() (NewEvent, error)

// Dispatch builds the event, then records and publishes it; failures are logged, never returned.
func (d Dispatcher) Dispatch(ctx context.Context, buildEvent EventBuilder) {
	event, err := buildEvent()
	if err != nil {
		slog.Error(buildEventFailedText, "error", err)
		return
	}

	recordEvent(ctx, d.Recorder, event)

	publishEvent(ctx, d.Publisher, event)
}

func recordEvent(ctx context.Context, recorder Recorder, event NewEvent) {
	if recorder == nil {
		slog.Error(recorderMissingText, "event", event.Label, "userId", event.UserID)
		return
	}

	if err := recorder(ctx, event); err != nil {
		slog.Error(recordEventFailedText,
			"event", event.Label, "userId", event.UserID, "error", err)
	}
}

func publishEvent(ctx context.Context, publisher Publisher, event NewEvent) {
	if publisher == nil {
		slog.Error(publisherMissingText, "event", event.Label, "userId", event.UserID)
		return
	}

	if err := publisher(ctx, event); err != nil {
		slog.Error(publishEventFailedText,
			"event", event.Label, "userId", event.UserID, "error", err)
	}
}
