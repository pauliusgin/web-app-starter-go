package events

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

const testLabel = Label("user.signed_up")

const confidentialLabel = Label("auth_challenge.generated")

const testChallengeToken = "9f1c4b7e2a0d48c6b3115ea77c0d5e93"

var fixedOccurredAt = time.Date(2026, time.September, 4, 9, 30, 0, 0, time.UTC)

type inMemoryEventLog struct {
	recordedEvents []NewEvent
	failure        error
}

func (l *inMemoryEventLog) Save(_ context.Context, newEvent NewEvent) (Event, error) {
	if l.failure != nil {
		return Event{}, l.failure
	}

	l.recordedEvents = append(l.recordedEvents, newEvent)

	return Event{
		ID:         newEvent.ID,
		UserID:     newEvent.UserID,
		Label:      newEvent.Label,
		Data:       newEvent.Data,
		OccurredAt: fixedOccurredAt,
	}, nil
}

func (l *inMemoryEventLog) onlyRecordedEvent(t *testing.T) NewEvent {
	t.Helper()

	if len(l.recordedEvents) != 1 {
		t.Fatalf("stored rows = %d, want 1", len(l.recordedEvents))
	}

	return l.recordedEvents[0]
}

func TestRecordStoresTheEventsOwnData(t *testing.T) {
	eventLog := &inMemoryEventLog{}
	accountEvent := NewEvent{
		ID:     uuid.New(),
		UserID: uuid.NullUUID{UUID: uuid.New(), Valid: true},
		Label:  testLabel,
		Data:   json.RawMessage(`{"email":"` + testEmail + `"}`),
	}

	recordedEvent, err := Record(context.Background(), RecordDependencies{EventWriteRepository: eventLog}, accountEvent)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	recordedRow := eventLog.onlyRecordedEvent(t)
	if recordedRow.Label != testLabel {
		t.Errorf("Label = %q, want %q", recordedRow.Label, testLabel)
	}
	if recordedRow.UserID != accountEvent.UserID {
		t.Errorf("UserID = %+v, want %+v", recordedRow.UserID, accountEvent.UserID)
	}
	if string(recordedRow.Data) != string(accountEvent.Data) {
		t.Errorf("Data = %s, want the event's own Data %s", recordedRow.Data, accountEvent.Data)
	}
	if recordedEvent.ID != accountEvent.ID {
		t.Errorf("ID = %s, want the id the event was dispatched with %s", recordedEvent.ID, accountEvent.ID)
	}
	if !recordedEvent.OccurredAt.Equal(fixedOccurredAt) {
		t.Errorf("OccurredAt = %s, want the stamp the repository returned %s",
			recordedEvent.OccurredAt, fixedOccurredAt)
	}
}

func TestRecordStoresTheRedactedPayloadInPlaceOfTheBrokerOne(t *testing.T) {
	eventLog := &inMemoryEventLog{}
	redactedPayload := `{"target":"` + testEmail + `"}`

	_, err := Record(context.Background(), RecordDependencies{EventWriteRepository: eventLog}, NewEvent{
		ID:           uuid.New(),
		Label:        confidentialLabel,
		Data:         json.RawMessage(`{"target":"` + testEmail + `","token":"` + testChallengeToken + `"}`),
		RedactedData: json.RawMessage(redactedPayload),
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	recordedRow := eventLog.onlyRecordedEvent(t)
	if string(recordedRow.Data) != redactedPayload {
		t.Errorf("Data = %s, want the event's RedactedData %s stored in its place", recordedRow.Data, redactedPayload)
	}
	if recordedRow.RedactedData != nil {
		t.Errorf("RedactedData = %s, want it emptied once it has replaced Data", recordedRow.RedactedData)
	}
}

func TestRecordSurfacesAStorageFailure(t *testing.T) {
	cause := errors.New("pq: relation \"event_log\" does not exist")
	eventLog := &inMemoryEventLog{failure: ErrRecordEventFailed{Label: testLabel, Cause: cause}}

	recordedEvent, err := Record(context.Background(), RecordDependencies{EventWriteRepository: eventLog}, newSignedUpForTest(t))

	if !errors.Is(err, cause) {
		t.Fatalf("error = %v, want it to unwrap to the driver error", err)
	}
	if recordedEvent.ID != uuid.Nil || recordedEvent.Data != nil {
		t.Errorf("event = %+v, want the zero value on failure", recordedEvent)
	}
}
