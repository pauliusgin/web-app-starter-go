package main

import (
	"context"

	eventsrepo "github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/events/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

type buildEventDispatcherParams struct {
	Queries   *sqlc.Queries
	Publisher events.Publisher
}

func buildEventDispatcher(params buildEventDispatcherParams) events.Dispatcher {
	recordEventDependencies := events.RecordDependencies{
		EventWriteRepository: eventsrepo.NewEventWriteRepository(params.Queries),
	}
	eventRecorder := func(ctx context.Context, event events.NewEvent) error {
		_, err := events.Record(ctx, recordEventDependencies, event)
		return err
	}

	return events.Dispatcher{Recorder: eventRecorder, Publisher: params.Publisher}
}
