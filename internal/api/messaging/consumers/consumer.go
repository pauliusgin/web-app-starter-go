// Package consumers declares which consumed events reach which usecases.
package consumers

import (
	"context"
	"time"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

// EventDelivery turns one delivered event body into a domain call, like http.HandlerFunc.
type EventDelivery func(ctx context.Context, body []byte) error

// Consumer binds one event label to the queue it is consumed from and the call it triggers.
type Consumer struct {
	Label          events.Label
	Queue          string
	Concurrency    int
	Prefetch       int
	HandlerTimeout time.Duration
	Handle         EventDelivery
}

// NewConsumerParams binds one event's payload type to the handler that consumes it.
type NewConsumerParams[Payload any] struct {
	Label          events.Label
	Queue          string
	Concurrency    int
	Prefetch       int
	HandlerTimeout time.Duration
	Handle         func(ctx context.Context, payload Payload) error
}

// NewConsumer declares the consumer that decodes Payload off its queue and hands it to Handle.
func NewConsumer[Payload any](params NewConsumerParams[Payload]) Consumer {
	consumer := Consumer{
		Label:          params.Label,
		Queue:          params.Queue,
		Concurrency:    params.Concurrency,
		Prefetch:       params.Prefetch,
		HandlerTimeout: params.HandlerTimeout,
	}

	if params.Handle == nil {
		return consumer
	}

	consumer.Handle = func(ctx context.Context, body []byte) error {
		payload, err := decodeEventPayload[Payload](decodeEventPayloadParams{
			Body:  body,
			Label: params.Label,
		})
		if err != nil {
			return err
		}

		return params.Handle(ctx, payload)
	}

	return consumer
}
