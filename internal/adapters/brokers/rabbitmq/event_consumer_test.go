package rabbitmq

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	rbmq "github.com/wagslane/go-rabbitmq"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

const testQueue = "web_app_starter.auth_challenge.email"

const testChallengeToken = "plaintext-token-that-must-never-be-logged"

// deliveryHandler answers every delivery with handleError, or panics with panicWith when set.
type deliveryHandler struct {
	handleError error
	panicWith   any
}

func (h deliveryHandler) handle(_ context.Context, _ []byte) error {
	if h.panicWith != nil {
		panic(h.panicWith)
	}

	return h.handleError
}

func newTestEventConsumer(handler deliveryHandler, logs *bytes.Buffer) EventConsumer {
	return EventConsumer{
		queue:          testQueue,
		handle:         handler.handle,
		handlerTimeout: defaultHandlerTimeout,
		logger:         slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
	}
}

func testDeliveryBody() []byte {
	return []byte(`{"challengeId":"6f1b1a3e-1d2c-4a5b-8c7d-9e0f1a2b3c4d",` +
		`"target":"accountant@example.com","purpose":"email_verification",` +
		`"token":"` + testChallengeToken + `"}`)
}

// acceptingAcknowledger stands in for the broker channel; requeueing is proven against a live broker in
// TestFailingEventIsRetriedThenDeadLettered.
type acceptingAcknowledger struct{}

func (acceptingAcknowledger) Ack(uint64, bool) error { return nil }

func (acceptingAcknowledger) Nack(uint64, bool, bool) error { return nil }

func (acceptingAcknowledger) Reject(uint64, bool) error { return nil }

func newTestDelivery(body []byte) rbmq.Delivery {
	return rbmq.Delivery{Delivery: amqp.Delivery{
		Acknowledger: acceptingAcknowledger{},
		Body:         body,
		Exchange:     EventExchange,
		RoutingKey:   string(events.AuthChallengeGeneratedLabel),
		Type:         string(events.AuthChallengeGeneratedLabel),
		MessageId:    "b3b2f1a0-0000-4000-8000-000000000000",
		ContentType:  eventContentType,
	}}
}

func TestConsumeDeliveryAcksSuccessAndHandsFailureBackForRetry(t *testing.T) {
	cases := []struct {
		name            string
		handle          func(ctx context.Context, body []byte) error
		wantAction      rbmq.Action
		wantLogContains string
	}{
		{
			name: "handled with the body untouched",
			handle: func(_ context.Context, body []byte) error {
				if !bytes.Equal(body, testDeliveryBody()) {
					return errors.New("body changed on its way to the handler")
				}
				return nil
			},
			wantAction: rbmq.Ack,
		},
		{
			name:            "handler reported a failure",
			handle:          deliveryHandler{handleError: errors.New("brevo refused it")}.handle,
			wantAction:      rbmq.Manual,
			wantLogContains: "event delivery not handled",
		},
		{
			name:            "handler panicked",
			handle:          deliveryHandler{panicWith: "nil map write"}.handle,
			wantAction:      rbmq.Manual,
			wantLogContains: "event handler panicked",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			logs := &bytes.Buffer{}
			consumer := newTestEventConsumer(deliveryHandler{}, logs)
			consumer.handle = testCase.handle

			action := consumer.consumeDelivery(newTestDelivery(testDeliveryBody()))

			if action != testCase.wantAction {
				t.Errorf("action = %v, want %v (logs %q)", action, testCase.wantAction, logs.String())
			}
			if testCase.wantLogContains != "" && !strings.Contains(logs.String(), testCase.wantLogContains) {
				t.Errorf("logs = %q, want them to mention %q", logs.String(), testCase.wantLogContains)
			}
		})
	}
}

func TestConsumeDeliveryNeverLogsTheBody(t *testing.T) {
	cases := []struct {
		name    string
		handler deliveryHandler
	}{
		{name: "success", handler: deliveryHandler{}},
		{name: "failure", handler: deliveryHandler{handleError: errors.New("send failed")}},
		{name: "panic", handler: deliveryHandler{panicWith: "boom"}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			logs := &bytes.Buffer{}
			consumer := newTestEventConsumer(testCase.handler, logs)

			consumer.consumeDelivery(newTestDelivery(testDeliveryBody()))

			if strings.Contains(logs.String(), testChallengeToken) {
				t.Errorf("logs = %q, want the plaintext token absent", logs.String())
			}
		})
	}
}

func TestNewEventConsumerRefusesAnUnusableDeclaration(t *testing.T) {
	handle := func(context.Context, []byte) error { return nil }
	routingKeys := []string{string(events.AuthChallengeGeneratedLabel)}

	cases := []struct {
		name    string
		params  NewEventConsumerParams
		wantErr error
	}{
		{
			name:    "no queue",
			params:  NewEventConsumerParams{RoutingKeys: routingKeys, Handle: handle},
			wantErr: ErrConsumerQueueMissing{},
		},
		{
			name:    "no routing keys",
			params:  NewEventConsumerParams{Queue: testQueue, Handle: handle},
			wantErr: ErrConsumerBindingMissing{Queue: testQueue},
		},
		{
			name:    "no handler",
			params:  NewEventConsumerParams{Queue: testQueue, RoutingKeys: routingKeys},
			wantErr: ErrConsumerHandlerMissing{Queue: testQueue},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			consumer, err := NewEventConsumer(testCase.params)

			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("error = %v, want %v", err, testCase.wantErr)
			}
			notWired := ErrConsumerNotWired{}
			if !errors.As(consumer.Run(), &notWired) {
				t.Errorf("the returned consumer ran, want the unusable zero value")
			}
		})
	}
}
