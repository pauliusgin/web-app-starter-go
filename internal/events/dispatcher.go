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

	recordEvent(ctx, recordEventParams{Recorder: d.Recorder, Event: event})

	publishEvent(ctx, publishEventParams{Publisher: d.Publisher, Event: event})
}

type recordEventParams struct {
	Recorder Recorder
	Event    NewEvent
}

func recordEvent(ctx context.Context, params recordEventParams) {
	event := params.Event
	if params.Recorder == nil {
		slog.Error(recorderMissingText, "event", event.Label, "userId", event.UserID)
		return
	}

	if err := params.Recorder(ctx, event); err != nil {
		slog.Error(recordEventFailedText,
			"event", event.Label, "userId", event.UserID, "error", err)
	}
}

type publishEventParams struct {
	Publisher Publisher
	Event     NewEvent
}

func publishEvent(ctx context.Context, params publishEventParams) {
	event := params.Event
	if params.Publisher == nil {
		slog.Error(publisherMissingText, "event", event.Label, "userId", event.UserID)
		return
	}

	if err := params.Publisher(ctx, event); err != nil {
		slog.Error(publishEventFailedText,
			"event", event.Label, "userId", event.UserID, "error", err)
	}
}
