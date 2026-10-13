package rabbitmq

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

const deadLetterArrivalTimeout = 10 * time.Second

func TestDeadLetterQueueNameSuffixesTheConsumerQueue(t *testing.T) {
	if got := DeadLetterQueueName(testQueue); got != testQueue+".dead" {
		t.Errorf("DeadLetterQueueName(%q) = %q, want %q", testQueue, got, testQueue+".dead")
	}
}

func TestDeclareDeadLetterTopologyRoutesEachQueueToItsDeadLetterQueue(t *testing.T) {
	amqpURL := liveRabbitMQURL()
	requireReachableBroker(t, amqpURL)

	queues := []string{
		"web_app_starter.test.topology." + uuid.NewString(),
		"web_app_starter.test.topology." + uuid.NewString(),
	}
	t.Cleanup(func() { deleteTestQueues(t, amqpURL, DeadLetterQueueName(queues[0]), DeadLetterQueueName(queues[1])) })

	err := DeclareDeadLetterTopology(DeclareDeadLetterTopologyParams{AmqpURL: amqpURL, Queues: queues})
	if err != nil {
		t.Fatalf("DeclareDeadLetterTopology = %v, want nil", err)
	}

	connection, channel := openTestChannel(t, amqpURL)
	defer connection.Close()

	for _, queue := range queues {
		err := channel.PublishWithContext(context.Background(), DeadLetterExchange, queue, false, false, amqp.Publishing{Body: []byte(queue)})
		if err != nil {
			t.Fatalf("Publish to %q with key %q: %v", DeadLetterExchange, queue, err)
		}

		deadLetter := awaitDeadLetter(t, amqpURL, DeadLetterQueueName(queue))
		if string(deadLetter.Body) != queue {
			t.Errorf("%q received %q, want the event dead-lettered from %q", DeadLetterQueueName(queue), deadLetter.Body, queue)
		}
	}
}

func TestDeclareDeadLetterTopologyReportsAnUnreachableBroker(t *testing.T) {
	unreachableURL := "amqp://user:password@" + closedLocalAddress(t) + "/web_app_starter"

	err := DeclareDeadLetterTopology(DeclareDeadLetterTopologyParams{
		AmqpURL: unreachableURL,
		Queues:  []string{testQueue},
	})

	if _, ok := errors.AsType[ErrBrokerUnreachable](err); !ok {
		t.Fatalf("DeclareDeadLetterTopology = %v, want ErrBrokerUnreachable", err)
	}
}

const liveTestRetryDelay = 200 * time.Millisecond

func TestFailingEventIsRetriedThenDeadLettered(t *testing.T) {
	amqpURL := liveRabbitMQURL()
	requireReachableBroker(t, amqpURL)

	queue := "web_app_starter.test.dead_letter." + uuid.NewString()
	label := events.Label("test.dead_letter." + uuid.NewString())
	t.Cleanup(func() { deleteTestQueues(t, amqpURL, queue, DeadLetterQueueName(queue)) })

	err := DeclareDeadLetterTopology(DeclareDeadLetterTopologyParams{AmqpURL: amqpURL, Queues: []string{queue}})
	if err != nil {
		t.Fatalf("DeclareDeadLetterTopology = %v", err)
	}

	connection, err := NewBrokerConnection(amqpURL, discardingLoggerForTest())
	if err != nil {
		t.Fatalf("NewBrokerConnection = %v", err)
	}
	t.Cleanup(func() { connection.Close() })

	publisher, err := NewEventPublisher(NewEventPublisherParams{Connection: connection, Logger: discardingLoggerForTest()})
	if err != nil {
		t.Fatalf("NewEventPublisher = %v", err)
	}
	t.Cleanup(publisher.Close)

	attempts := atomic.Int32{}
	consumer, err := NewEventConsumer(NewEventConsumerParams{
		Connection:  connection,
		Logger:      discardingLoggerForTest(),
		Queue:       queue,
		RoutingKeys: []string{string(label)},
		RetryDelay:  liveTestRetryDelay,
		Handle: func(context.Context, []byte) error {
			attempts.Add(1)
			return errors.New("brevo is down")
		},
	})
	if err != nil {
		t.Fatalf("NewEventConsumer = %v", err)
	}
	running := StartEventConsumer(consumer)
	t.Cleanup(func() { StopEventConsumer(running, time.Second) })

	awaitQueueConsumed(t, amqpURL, queue)

	publishedAt := time.Now()
	err = publisher.PublishEvent(context.Background(), events.NewEvent{Label: label, Data: []byte(`{"n":1}`)})
	if err != nil {
		t.Fatalf("PublishEvent = %v", err)
	}

	deadLetter := awaitDeadLetter(t, amqpURL, DeadLetterQueueName(queue))
	elapsed := time.Since(publishedAt)

	if string(deadLetter.Body) != `{"n":1}` {
		t.Errorf("dead-lettered body = %q, want the published one", deadLetter.Body)
	}
	if got := attempts.Load(); got != maxDeliveryRetries+1 {
		t.Errorf("handler attempts = %d, want %d (one try plus %d retries)", got, maxDeliveryRetries+1, maxDeliveryRetries)
	}
	if minimum := maxDeliveryRetries * liveTestRetryDelay; elapsed < minimum {
		t.Errorf("dead-lettered after %s, want at least %s of retry delays", elapsed, minimum)
	}
}

func openTestChannel(t *testing.T, amqpURL string) (*amqp.Connection, *amqp.Channel) {
	t.Helper()

	connection, err := amqp.Dial(amqpURL)
	if err != nil {
		t.Fatalf("amqp.Dial: %v", err)
	}
	channel, err := connection.Channel()
	if err != nil {
		connection.Close()
		t.Fatalf("Channel: %v", err)
	}

	return connection, channel
}

func awaitQueueConsumed(t *testing.T, amqpURL string, queue string) {
	t.Helper()

	deadline := time.Now().Add(deadLetterArrivalTimeout)
	for time.Now().Before(deadline) {
		connection, channel := openTestChannel(t, amqpURL)
		state, err := channel.QueueDeclarePassive(queue, true, false, false, false, nil)
		connection.Close()
		if err == nil && state.Consumers > 0 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("no consumer attached to %q within %s", queue, deadLetterArrivalTimeout)
}

func awaitDeadLetter(t *testing.T, amqpURL string, deadLetterQueue string) amqp.Delivery {
	t.Helper()

	connection, channel := openTestChannel(t, amqpURL)
	defer connection.Close()

	deadline := time.Now().Add(deadLetterArrivalTimeout)
	for time.Now().Before(deadline) {
		delivery, found, err := channel.Get(deadLetterQueue, true)
		if err != nil {
			t.Fatalf("Get(%q): %v", deadLetterQueue, err)
		}
		if found {
			return delivery
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("nothing reached %q within %s", deadLetterQueue, deadLetterArrivalTimeout)
	return amqp.Delivery{}
}

func deleteTestQueues(t *testing.T, amqpURL string, queues ...string) {
	t.Helper()

	connection, channel := openTestChannel(t, amqpURL)
	defer connection.Close()

	for _, queue := range queues {
		if _, err := channel.QueueDelete(queue, false, false, false); err != nil {
			t.Errorf("QueueDelete(%q): %v", queue, err)
		}
	}
}
