package rabbitmq

import (
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	rbmq "github.com/wagslane/go-rabbitmq"
)

// DeadLetterExchange receives every event a consumer queue gave up on after its retries.
const DeadLetterExchange = "web_app_starter.events.dead"

const deadLetterExchangeKind = "direct"

const deadLetterQueueSuffix = ".dead"

const deadLetterRetention = 7 * 24 * time.Hour

const deadLetterMaxLength = 10_000

const maxDeliveryRetries = 3

// DeadLetterQueueName names the queue holding what the given consumer queue dead-lettered.
func DeadLetterQueueName(queue string) string {
	return queue + deadLetterQueueSuffix
}

// DeclareDeadLetterTopologyParams names the broker and the consumer queues that need a dead-letter queue.
type DeclareDeadLetterTopologyParams struct {
	AmqpURL string
	Queues  []string
}

// DeclareDeadLetterTopology declares the dead-letter exchange and one bound dead-letter queue per queue.
func DeclareDeadLetterTopology(params DeclareDeadLetterTopologyParams) error {
	connection, err := amqp.Dial(params.AmqpURL)
	if err != nil {
		return ErrBrokerUnreachable{Cause: err}
	}
	defer connection.Close()

	channel, err := connection.Channel()
	if err != nil {
		return ErrDeadLetterExchangeUndeclared{Cause: err}
	}
	defer channel.Close()

	err = channel.ExchangeDeclare(DeadLetterExchange, deadLetterExchangeKind, true, false, false, false, nil)
	if err != nil {
		return ErrDeadLetterExchangeUndeclared{Cause: err}
	}

	for _, queue := range params.Queues {
		if err := declareDeadLetterQueue(channel, queue); err != nil {
			return err
		}
	}

	return nil
}

func declareDeadLetterQueue(channel *amqp.Channel, queue string) error {
	deadLetterQueue := DeadLetterQueueName(queue)

	_, err := channel.QueueDeclare(deadLetterQueue, true, false, false, false, deadLetterQueueArgs())
	if err != nil {
		return ErrDeadLetterQueueUndeclared{Queue: deadLetterQueue, Cause: err}
	}

	err = channel.QueueBind(deadLetterQueue, queue, DeadLetterExchange, false, nil)
	if err != nil {
		return ErrDeadLetterQueueUndeclared{Queue: deadLetterQueue, Cause: err}
	}

	return nil
}

func deadLetterQueueArgs() amqp.Table {
	return amqp.Table{
		"x-message-ttl": deadLetterRetention.Milliseconds(),
		"x-max-length":  int64(deadLetterMaxLength),
	}
}

func retryQueueArgs(params NewEventConsumerParams) rbmq.Table {
	retryDelay := params.RetryDelay.Milliseconds()

	return rbmq.Table{
		"x-queue-type":              "quorum",
		"x-delivery-limit":          int64(maxDeliveryRetries),
		"x-dead-letter-exchange":    DeadLetterExchange,
		"x-dead-letter-routing-key": params.Queue,
		"x-delayed-retry-type":      "failed",
		"x-delayed-retry-min":       retryDelay,
		"x-delayed-retry-max":       retryDelay,
	}
}
