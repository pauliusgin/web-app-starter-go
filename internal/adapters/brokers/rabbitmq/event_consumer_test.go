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

	"github.com/pauliusgin/web_app_starter-be/internal/config"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

const testQueue = "web_app_starter.auth_challenge.email"

const testChallengeToken = "plaintext-token-that-must-never-be-logged"

type recordingDeliveryHandler struct {
	received    [][]byte
	handleError error
	panicWith   any
}

func (h *recordingDeliveryHandler) handle(_ context.Context, body []byte) error {
	if h.panicWith != nil {
		panic(h.panicWith)
	}

	h.received = append(h.received, body)

	return h.handleError
}

func newTestEventConsumer(handler *recordingDeliveryHandler, logs *bytes.Buffer) EventConsumer {
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

type recordingAcknowledger struct {
	rejectedWithRequeue int
	rejectError         error
}

func (a *recordingAcknowledger) Ack(uint64, bool) error {
	return nil
}

func (a *recordingAcknowledger) Nack(uint64, bool, bool) error {
	return nil
}

func (a *recordingAcknowledger) Reject(_ uint64, requeue bool) error {
	if requeue {
		a.rejectedWithRequeue++
	}

	return a.rejectError
}

func newTestDelivery(body []byte) rbmq.Delivery {
	return newTestDeliveryOn(&recordingAcknowledger{}, body)
}

func newTestDeliveryOn(acknowledger amqp.Acknowledger, body []byte) rbmq.Delivery {
	return rbmq.Delivery{Delivery: amqp.Delivery{
		Acknowledger: acknowledger,
		Body:         body,
		Exchange:     EventExchange,
		RoutingKey:   string(events.AuthChallengeGeneratedLabel),
		Type:         string(events.AuthChallengeGeneratedLabel),
		MessageId:    "b3b2f1a0-0000-4000-8000-000000000000",
		ContentType:  eventContentType,
	}}
}

func TestConsumeDeliveryPassesTheBodyOnAndAcknowledges(t *testing.T) {
	handler := &recordingDeliveryHandler{}
	consumer := newTestEventConsumer(handler, &bytes.Buffer{})

	action := consumer.consumeDelivery(newTestDelivery(testDeliveryBody()))

	if action != rbmq.Ack {
		t.Errorf("action = %v, want rbmq.Ack", action)
	}
	if len(handler.received) != 1 {
		t.Fatalf("handled deliveries = %d, want 1", len(handler.received))
	}
	if !bytes.Equal(handler.received[0], testDeliveryBody()) {
		t.Errorf("body = %q, want it handed on untouched", handler.received[0])
	}
}

func TestConsumeDeliveryAcksSuccessAndRequeuesFailure(t *testing.T) {
	cases := []struct {
		name            string
		handler         *recordingDeliveryHandler
		wantAction      rbmq.Action
		wantRejects     int
		wantHandled     int
		wantLogContains string
	}{
		{
			name:        "handled",
			handler:     &recordingDeliveryHandler{},
			wantAction:  rbmq.Ack,
			wantHandled: 1,
		},
		{
			name:            "handler reported a failure",
			handler:         &recordingDeliveryHandler{handleError: errors.New("brevo refused it")},
			wantAction:      rbmq.Manual,
			wantRejects:     1,
			wantHandled:     1,
			wantLogContains: "event delivery not handled",
		},
		{
			name:            "handler panicked",
			handler:         &recordingDeliveryHandler{panicWith: "nil map write"},
			wantAction:      rbmq.Manual,
			wantRejects:     1,
			wantHandled:     0,
			wantLogContains: "event handler panicked",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			logs := &bytes.Buffer{}
			consumer := newTestEventConsumer(testCase.handler, logs)
			acknowledger := &recordingAcknowledger{}

			action := consumer.consumeDelivery(newTestDeliveryOn(acknowledger, testDeliveryBody()))

			if action != testCase.wantAction {
				t.Errorf("action = %v, want %v", action, testCase.wantAction)
			}
			if acknowledger.rejectedWithRequeue != testCase.wantRejects {
				t.Errorf("rejects with requeue = %d, want %d",
					acknowledger.rejectedWithRequeue, testCase.wantRejects)
			}
			if len(testCase.handler.received) != testCase.wantHandled {
				t.Errorf("handled deliveries = %d, want %d",
					len(testCase.handler.received), testCase.wantHandled)
			}
			if testCase.wantLogContains == "" {
				return
			}
			if !strings.Contains(logs.String(), testCase.wantLogContains) {
				t.Errorf("logs = %q, want them to mention %q", logs.String(), testCase.wantLogContains)
			}
		})
	}
}

func TestConsumeDeliveryNeverLogsTheBody(t *testing.T) {
	cases := []struct {
		name    string
		handler *recordingDeliveryHandler
	}{
		{name: "success", handler: &recordingDeliveryHandler{}},
		{name: "failure", handler: &recordingDeliveryHandler{handleError: errors.New("send failed")}},
		{name: "panic", handler: &recordingDeliveryHandler{panicWith: "boom"}},
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

func TestConsumerErrorsUnwrapToTheirCause(t *testing.T) {
	cause := errors.New("Exception (404) Reason: \"NOT_FOUND - no exchange 'web_app_starter.events'\"")

	cases := []struct {
		name string
		err  error
	}{
		{name: "consumer unavailable", err: ErrConsumerUnavailable{Queue: testQueue, Cause: cause}},
		{name: "consumer start failed", err: ErrConsumerStartFailed{Queue: testQueue, Cause: cause}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if !errors.Is(testCase.err, cause) {
				t.Errorf("errors.Is did not recover %v from %v", cause, testCase.err)
			}
		})
	}
}

func TestStopAllEventConsumersToleratesAnEmptySet(t *testing.T) {
	StopAllEventConsumers(RunningEventConsumers{}, config.ConsumerStopTimeout)
}
