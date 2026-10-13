// Package rabbitmq publishes and consumes domain events on a RabbitMQ broker.
package rabbitmq

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	rbmq "github.com/wagslane/go-rabbitmq"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

// EventExchange is the topic exchange every domain event is published to.
const EventExchange = "web_app_starter.events"

const eventExchangeKind = "topic"

const eventContentType = "application/json"

const publishConfirmTimeout = 2 * time.Second

// EventPublisher publishes domain events on the broker's event exchange.
type EventPublisher struct {
	publisher *rbmq.Publisher
}

// NewEventPublisherParams carries what a publisher needs to open its channel.
type NewEventPublisherParams struct {
	Connection *rbmq.Conn
	Logger     *slog.Logger
}

// NewEventPublisher opens a confirming channel and declares the durable topic exchange.
func NewEventPublisher(params NewEventPublisherParams) (EventPublisher, error) {
	publisher, err := rbmq.NewPublisher(
		params.Connection,
		rbmq.WithPublisherOptionsLogger(newBrokerLogger(params.Logger)),
		rbmq.WithPublisherOptionsExchangeName(EventExchange),
		rbmq.WithPublisherOptionsExchangeKind(eventExchangeKind),
		rbmq.WithPublisherOptionsExchangeDurable,
		rbmq.WithPublisherOptionsExchangeDeclare,
		rbmq.WithPublisherOptionsConfirm,
	)
	if err != nil {
		return EventPublisher{}, ErrPublisherUnavailable{Cause: err}
	}

	return EventPublisher{publisher: publisher}, nil
}

// PublishEvent publishes one event and blocks until the broker resolves it.
func (p EventPublisher) PublishEvent(ctx context.Context, event events.NewEvent) error {
	if p.publisher == nil {
		return ErrPublisherNotWired{Label: event.Label}
	}

	message := buildEventMessage(event)

	confirmations, err := p.publisher.PublishWithDeferredConfirmWithContext(
		ctx,
		message.body,
		[]string{message.routingKey},
		rbmq.WithPublishOptionsExchange(EventExchange),
		rbmq.WithPublishOptionsContentType(eventContentType),
		rbmq.WithPublishOptionsPersistentDelivery,
		rbmq.WithPublishOptionsType(string(event.Label)),
		rbmq.WithPublishOptionsMessageID(uuid.NewString()),
		rbmq.WithPublishOptionsTimestamp(time.Now().UTC()),
	)
	if err != nil {
		return ErrPublishFailed{Label: event.Label, RoutingKey: message.routingKey, Cause: err}
	}

	return awaitPublishConfirmation(ctx, confirmations, message)
}

// Close stops the publisher, and must be called before closing the connection it was built on.
func (p EventPublisher) Close() {
	if p.publisher == nil {
		return
	}

	p.publisher.Close()
}

type eventMessage struct {
	label      events.Label
	routingKey string
	body       []byte
}

func buildEventMessage(event events.NewEvent) eventMessage {
	return eventMessage{
		label:      event.Label,
		routingKey: string(event.Label),
		body:       event.Data,
	}
}

func awaitPublishConfirmation(ctx context.Context, confirmations rbmq.PublisherConfirmation, message eventMessage) error {
	if len(confirmations) != 1 || confirmations[0] == nil {
		return ErrConfirmUnavailable{Label: message.label}
	}

	confirmCtx, cancelWait := context.WithTimeout(context.WithoutCancel(ctx), publishConfirmTimeout)
	defer cancelWait()

	acknowledged, err := confirmations[0].WaitContext(confirmCtx)
	if err != nil {
		return ErrPublishConfirmTimedOut{Label: message.label, Timeout: publishConfirmTimeout, Cause: err}
	}
	if !acknowledged {
		return ErrPublishNacked{Label: message.label, RoutingKey: message.routingKey}
	}

	return nil
}
