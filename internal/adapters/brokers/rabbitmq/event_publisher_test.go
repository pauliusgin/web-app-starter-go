package rabbitmq

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/user"
)

const testEmail = "accountant@example.com"

const testLabel events.Label = "payment.received"

func labelledEventForTest(label events.Label) events.NewEvent {
	return events.NewEvent{
		ID:    uuid.New(),
		Label: label,
		Data:  json.RawMessage(`{"reference":"INV-2026-0001"}`),
	}
}

func newSignedUpForTest(t *testing.T) events.NewEvent {
	t.Helper()

	signedUp, err := user.NewUserSignedUp(user.NewUserSignedUpParams{
		UserID: uuid.New(),
		Email:  testEmail,
	})
	if err != nil {
		t.Fatalf("NewUserSignedUp: %v", err)
	}

	return signedUp
}

func TestBuildEventMessageRoutesOnTheEventLabel(t *testing.T) {
	cases := []struct {
		name           string
		event          events.NewEvent
		wantRoutingKey string
	}{
		{name: "a signup", event: newSignedUpForTest(t), wantRoutingKey: "user.signed_up"},
		{name: "any other event", event: labelledEventForTest(testLabel), wantRoutingKey: "payment.received"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			message := buildEventMessage(testCase.event)

			if message.routingKey != testCase.wantRoutingKey {
				t.Errorf("routingKey = %q, want %q", message.routingKey, testCase.wantRoutingKey)
			}
			if message.routingKey != string(testCase.event.Label) {
				t.Errorf("routingKey = %q, want the event label %q", message.routingKey, testCase.event.Label)
			}
			if message.label != testCase.event.Label {
				t.Errorf("label = %q, want %q", message.label, testCase.event.Label)
			}
		})
	}
}

func TestBuildEventMessageCarriesExactlyTheEventPayload(t *testing.T) {
	signedUp := newSignedUpForTest(t)

	message := buildEventMessage(signedUp)

	if !bytes.Equal(message.body, signedUp.Data) {
		t.Fatalf("body = %s, want the event's own payload %s", message.body, signedUp.Data)
	}

	payload := map[string]any{}
	if err := json.Unmarshal(message.body, &payload); err != nil {
		t.Fatalf("the published body %s is not valid JSON: %v", message.body, err)
	}
	if len(payload) != 2 {
		t.Errorf("payload = %s, want only the keys Data carries and no envelope around them", message.body)
	}
	for _, key := range []string{"userId", "email"} {
		if _, isPresent := payload[key]; !isPresent {
			t.Errorf("payload %s has no %q key, want it present", message.body, key)
		}
	}
}

func TestPublishEventRefusesTheZeroValuePublisher(t *testing.T) {
	publisher := EventPublisher{}
	signedUp := newSignedUpForTest(t)

	err := publisher.PublishEvent(context.Background(), signedUp)

	publisherNotWired := ErrPublisherNotWired{}
	if !errors.As(err, &publisherNotWired) {
		t.Fatalf("PublishEvent = %v, want ErrPublisherNotWired rather than a panic", err)
	}
	if publisherNotWired.Label != events.UserSignedUpLabel {
		t.Errorf("Label = %q, want %q", publisherNotWired.Label, events.UserSignedUpLabel)
	}
}

func TestPublisherErrorsUnwrapToTheirCause(t *testing.T) {
	cause := errors.New("dial tcp 127.0.0.1:5672: connect: connection refused")

	cases := []struct {
		name      string
		err       error
		wantCause error
	}{
		{name: "broker unreachable", err: ErrBrokerUnreachable{Cause: cause}, wantCause: cause},
		{name: "publisher unavailable", err: ErrPublisherUnavailable{Cause: cause}, wantCause: cause},
		{
			name:      "publish failed",
			err:       ErrPublishFailed{Label: testLabel, RoutingKey: string(testLabel), Cause: cause},
			wantCause: cause,
		},
		{
			name:      "confirm timed out",
			err:       ErrPublishConfirmTimedOut{Label: testLabel, Timeout: 2 * time.Second, Cause: context.DeadlineExceeded},
			wantCause: context.DeadlineExceeded,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if !errors.Is(testCase.err, testCase.wantCause) {
				t.Errorf("errors.Is did not recover %v from %v", testCase.wantCause, testCase.err)
			}
		})
	}
}
