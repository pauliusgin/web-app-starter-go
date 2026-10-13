package events

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"
)

const testEmail = "accountant@example.com"

type recordingLog struct {
	recordedEvents []NewEvent
	failure        error
}

func (l *recordingLog) recorder(_ context.Context, event NewEvent) error {
	if l.failure != nil {
		return l.failure
	}

	l.recordedEvents = append(l.recordedEvents, event)

	return nil
}

type recordingBroker struct {
	publishedEvents []NewEvent
	failure         error
}

func (b *recordingBroker) publisher(_ context.Context, event NewEvent) error {
	if b.failure != nil {
		return b.failure
	}

	b.publishedEvents = append(b.publishedEvents, event)

	return nil
}

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	logs := &bytes.Buffer{}
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	return logs
}

func newSignedUpForTest(t *testing.T) NewEvent {
	t.Helper()

	userID := uuid.New()

	signedUp, err := New(NewEventParams{
		Label:  UserSignedUpLabel,
		UserID: uuid.NullUUID{UUID: userID, Valid: true},
		Data: map[string]any{
			"userId": userID,
			"email":  testEmail,
		},
	})
	if err != nil {
		t.Fatalf("New event: %v", err)
	}

	return signedUp
}

func builtEvent(event NewEvent) EventBuilder {
	return func() (NewEvent, error) {
		return event, nil
	}
}

func TestDispatchRecordsAndPublishesTheEvent(t *testing.T) {
	logs := captureLogs(t)
	log := &recordingLog{}
	broker := &recordingBroker{}
	signedUp := newSignedUpForTest(t)

	Dispatcher{Recorder: log.recorder, Publisher: broker.publisher}.Dispatch(context.Background(), builtEvent(signedUp))

	if len(log.recordedEvents) != 1 {
		t.Fatalf("recorded events = %d, want exactly the one event dispatched", len(log.recordedEvents))
	}
	if len(broker.publishedEvents) != 1 {
		t.Fatalf("published events = %d, want exactly the one event dispatched", len(broker.publishedEvents))
	}
	for _, handedOver := range []NewEvent{log.recordedEvents[0], broker.publishedEvents[0]} {
		if handedOver.Label != UserSignedUpLabel {
			t.Errorf("Label = %q, want %q", handedOver.Label, UserSignedUpLabel)
		}
		if handedOver.UserID != signedUp.UserID {
			t.Errorf("UserID = %+v, want %+v", handedOver.UserID, signedUp.UserID)
		}
	}
	if logs.Len() != 0 {
		t.Errorf("logs = %s, want nothing logged for a dispatched event", logs)
	}
}

func TestDispatchStillPublishesWhenRecordingFails(t *testing.T) {
	logs := captureLogs(t)
	cause := errors.New("the event log is unreachable")
	log := &recordingLog{failure: cause}
	broker := &recordingBroker{}
	signedUp := newSignedUpForTest(t)

	Dispatcher{Recorder: log.recorder, Publisher: broker.publisher}.Dispatch(context.Background(), builtEvent(signedUp))

	if len(log.recordedEvents) != 0 {
		t.Errorf("recorded events = %d, want none from a refusing recorder", len(log.recordedEvents))
	}
	if len(broker.publishedEvents) != 1 {
		t.Errorf("published events = %d, want the event published despite the recording failure", len(broker.publishedEvents))
	}
	assertLogsCarry(t, logs.String(), signedUp, "failed recording the event", cause.Error())
}

func TestDispatchSwallowsAPublisherFailureAndLogsIt(t *testing.T) {
	logs := captureLogs(t)
	cause := errors.New("the broker is unreachable")
	log := &recordingLog{}
	broker := &recordingBroker{failure: cause}
	signedUp := newSignedUpForTest(t)

	Dispatcher{Recorder: log.recorder, Publisher: broker.publisher}.Dispatch(context.Background(), builtEvent(signedUp))

	if len(log.recordedEvents) != 1 {
		t.Errorf("recorded events = %d, want the event recorded despite the publishing failure", len(log.recordedEvents))
	}
	if len(broker.publishedEvents) != 0 {
		t.Errorf("published events = %d, want none from a refusing broker", len(broker.publishedEvents))
	}
	assertLogsCarry(t, logs.String(), signedUp, "failed publishing the event", cause.Error())
}

func TestDispatchLogsLoudlyWhenNoRecorderIsWired(t *testing.T) {
	logs := captureLogs(t)
	broker := &recordingBroker{}
	signedUp := newSignedUpForTest(t)

	Dispatcher{Publisher: broker.publisher}.Dispatch(context.Background(), builtEvent(signedUp))

	if len(broker.publishedEvents) != 1 {
		t.Errorf("published events = %d, want the event published without a recorder", len(broker.publishedEvents))
	}
	assertLogsCarry(t, logs.String(), signedUp, "no event recorder is wired", "level=ERROR")
}

func TestDispatchLogsLoudlyWhenNoPublisherIsWired(t *testing.T) {
	logs := captureLogs(t)
	log := &recordingLog{}
	signedUp := newSignedUpForTest(t)

	Dispatcher{Recorder: log.recorder}.Dispatch(context.Background(), builtEvent(signedUp))

	if len(log.recordedEvents) != 1 {
		t.Errorf("recorded events = %d, want the event recorded without a publisher", len(log.recordedEvents))
	}
	assertLogsCarry(t, logs.String(), signedUp, "no event publisher is wired", "level=ERROR")
}

func TestDispatchDropsAnEventThatFailedToBuildAndLogsIt(t *testing.T) {
	logs := captureLogs(t)
	log := &recordingLog{}
	broker := &recordingBroker{}
	unbuildable := func() (NewEvent, error) {
		return New(NewEventParams{Label: UserSignedUpLabel, Data: func() {}})
	}

	Dispatcher{Recorder: log.recorder, Publisher: broker.publisher}.Dispatch(context.Background(), unbuildable)

	if len(log.recordedEvents) != 0 {
		t.Errorf("recorded events = %d, want none for an event that failed to build", len(log.recordedEvents))
	}
	if len(broker.publishedEvents) != 0 {
		t.Errorf("published events = %d, want none for an event that failed to build", len(broker.publishedEvents))
	}
	for _, want := range []string{"failed building the event", string(UserSignedUpLabel), "level=ERROR"} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("logs = %s, want them to carry %q", logs, want)
		}
	}
}

func assertLogsCarry(t *testing.T, logged string, event NewEvent, wants ...string) {
	t.Helper()

	wants = append(wants, string(event.Label), event.UserID.UUID.String())

	for _, want := range wants {
		if !strings.Contains(logged, want) {
			t.Errorf("logs = %s, want them to carry %q", logged, want)
		}
	}
}
