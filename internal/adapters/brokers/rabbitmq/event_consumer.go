package rabbitmq

import (
	"context"
	"log/slog"
	"sync"
	"time"

	rbmq "github.com/wagslane/go-rabbitmq"
)

// EventDeliveryHandler is what a consumer does with one delivered event body.
type EventDeliveryHandler func(ctx context.Context, body []byte) error

const defaultConsumerConcurrency = 1

const defaultConsumerPrefetch = 10

const defaultHandlerTimeout = 20 * time.Second

const defaultRetryDelay = 5 * time.Second

// EventConsumer consumes one durable queue bound to the broker's event exchange.
type EventConsumer struct {
	consumer       *rbmq.Consumer
	queue          string
	handle         EventDeliveryHandler
	handlerTimeout time.Duration
	logger         *slog.Logger
}

// NewEventConsumerParams carries the queue a consumer serves, what it binds, and how it is tuned.
type NewEventConsumerParams struct {
	Connection     *rbmq.Conn
	Logger         *slog.Logger
	Queue          string
	RoutingKeys    []string
	Concurrency    int
	Prefetch       int
	HandlerTimeout time.Duration
	RetryDelay     time.Duration
	Handle         EventDeliveryHandler
}

// NewEventConsumer opens a channel and prepares a consumer that is not yet running.
func NewEventConsumer(params NewEventConsumerParams) (EventConsumer, error) {
	if params.Queue == "" {
		return EventConsumer{}, ErrConsumerQueueMissing{}
	}
	if len(params.RoutingKeys) == 0 {
		return EventConsumer{}, ErrConsumerBindingMissing{Queue: params.Queue}
	}
	if params.Handle == nil {
		return EventConsumer{}, ErrConsumerHandlerMissing{Queue: params.Queue}
	}

	tuned := withConsumerDefaults(params)

	consumer, err := rbmq.NewConsumer(tuned.Connection, tuned.Queue, consumerChannelOptions(tuned)...)
	if err != nil {
		return EventConsumer{}, ErrConsumerUnavailable{Queue: tuned.Queue, Cause: err}
	}

	return EventConsumer{
		consumer:       consumer,
		queue:          tuned.Queue,
		handle:         tuned.Handle,
		handlerTimeout: tuned.HandlerTimeout,
		logger:         tuned.Logger,
	}, nil
}

// Run declares the queue and its bindings, then consumes until Close is called. It blocks.
func (c EventConsumer) Run() error {
	if c.consumer == nil {
		return ErrConsumerNotWired{Queue: c.queue}
	}

	if err := c.consumer.Run(c.consumeDelivery); err != nil {
		return ErrConsumerStartFailed{Queue: c.queue, Cause: err}
	}

	return nil
}

// CloseWithContext stops consuming, waiting for the in-flight handler until ctx is done.
func (c EventConsumer) CloseWithContext(ctx context.Context) {
	if c.consumer == nil {
		return
	}

	c.consumer.CloseWithContext(ctx)
}

func withConsumerDefaults(params NewEventConsumerParams) NewEventConsumerParams {
	if params.Logger == nil {
		params.Logger = slog.Default()
	}
	if params.Concurrency < 1 {
		params.Concurrency = defaultConsumerConcurrency
	}
	if params.Prefetch < 1 {
		params.Prefetch = defaultConsumerPrefetch
	}
	if params.HandlerTimeout <= 0 {
		params.HandlerTimeout = defaultHandlerTimeout
	}
	if params.RetryDelay < time.Millisecond {
		params.RetryDelay = defaultRetryDelay
	}

	return params
}

func consumerChannelOptions(params NewEventConsumerParams) []func(*rbmq.ConsumerOptions) {
	options := []func(*rbmq.ConsumerOptions){
		rbmq.WithConsumerOptionsLogger(newBrokerLogger(params.Logger)),
		rbmq.WithConsumerOptionsQueueDurable,
		rbmq.WithConsumerOptionsQueueArgs(retryQueueArgs(params)),
		rbmq.WithConsumerOptionsExchangeName(EventExchange),
		rbmq.WithConsumerOptionsConcurrency(params.Concurrency),
		rbmq.WithConsumerOptionsQOSPrefetch(params.Prefetch),
	}

	for _, routingKey := range params.RoutingKeys {
		options = append(options, rbmq.WithConsumerOptionsRoutingKey(routingKey))
	}

	return options
}

func (c EventConsumer) consumeDelivery(delivery rbmq.Delivery) (action rbmq.Action) {
	defer c.recoverDeliveryPanic(delivery, &action)

	handlerCtx, cancelHandler := context.WithTimeout(context.Background(), c.handlerTimeout)
	defer cancelHandler()

	if err := c.handle(handlerCtx, delivery.Body); err != nil {
		c.logger.Error("event delivery not handled",
			append(deliveryLogAttributes(c.queue, delivery), "error", err)...)

		return c.rejectForRetry(delivery)
	}

	return rbmq.Ack
}

func (c EventConsumer) recoverDeliveryPanic(delivery rbmq.Delivery, action *rbmq.Action) {
	panicValue := recover()
	if panicValue == nil {
		return
	}

	c.logger.Error("event handler panicked",
		append(deliveryLogAttributes(c.queue, delivery), "panic", panicValue)...)

	*action = c.rejectForRetry(delivery)
}

func (c EventConsumer) rejectForRetry(delivery rbmq.Delivery) rbmq.Action {
	if err := delivery.Reject(true); err != nil {
		c.logger.Error("event delivery not returned for retry",
			append(deliveryLogAttributes(c.queue, delivery), "error", err)...)
	}

	return rbmq.Manual
}

func deliveryLogAttributes(queue string, delivery rbmq.Delivery) []any {
	return []any{
		"queue", queue,
		"routing_key", delivery.RoutingKey,
		"message_id", delivery.MessageId,
		"failed_deliveries", failedDeliveryCount(delivery),
		"max_retries", maxDeliveryRetries,
	}
}

func failedDeliveryCount(delivery rbmq.Delivery) int64 {
	switch count := delivery.Headers["x-delivery-count"].(type) {
	case int64:
		return count
	case int32:
		return int64(count)
	default:
		return 0
	}
}

// RunningEventConsumer is a started consumer plus the one-shot result of its Run call.
type RunningEventConsumer struct {
	consumer  EventConsumer
	runResult <-chan error
}

// StartEventConsumer hands Run its own goroutine and returns once it is under way.
func StartEventConsumer(consumer EventConsumer) RunningEventConsumer {
	runResult := make(chan error, 1)

	go func() {
		runResult <- consumer.Run()
	}()

	return RunningEventConsumer{consumer: consumer, runResult: runResult}
}

// StopEventConsumer closes the consumer and reports whatever Run finally returned.
func StopEventConsumer(running RunningEventConsumer, stopTimeout time.Duration) {
	logger := running.consumer.logger
	if logger == nil {
		logger = slog.Default()
	}

	queue := running.consumer.queue

	stopCtx, cancelStop := context.WithTimeout(context.Background(), stopTimeout)
	defer cancelStop()

	running.consumer.CloseWithContext(stopCtx)

	select {
	case err := <-running.runResult:
		if err != nil {
			logger.Error("event consumer stopped with an error", "queue", queue, "error", err)
			return
		}
	case <-time.After(stopTimeout):
		logger.Error("event consumer did not stop in time", "queue", queue, "timeout", stopTimeout)
		return
	}

	logger.Info("event consumer stopped", "queue", queue)
}

// RunningEventConsumers is every consumer one process started, in start order.
type RunningEventConsumers []RunningEventConsumer

// StopAllEventConsumers stops every consumer at once and waits for the slowest.
func StopAllEventConsumers(running RunningEventConsumers, stopTimeout time.Duration) {
	stopped := sync.WaitGroup{}

	for _, consumer := range running {
		stopped.Add(1)

		go func() {
			defer stopped.Done()
			StopEventConsumer(consumer, stopTimeout)
		}()
	}

	stopped.Wait()
}
