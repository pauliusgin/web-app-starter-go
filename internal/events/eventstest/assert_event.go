// Package eventstest checks built domain events against the payload contract subscribers rely on.
package eventstest

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

// WantEvent is what a built event must carry; a nil RedactedPayload means the event has no redacted copy.
type WantEvent struct {
	Label           events.Label
	UserID          uuid.UUID
	Payload         map[string]any
	RedactedPayload map[string]any
}

// AssertEvent fails the test unless the event carries exactly the wanted label, owner and payloads.
func AssertEvent(t testing.TB, event events.NewEvent, want WantEvent) {
	t.Helper()

	if event.Label != want.Label {
		t.Errorf("Label = %q, want %q", event.Label, want.Label)
	}
	if event.UserID != (uuid.NullUUID{UUID: want.UserID, Valid: true}) {
		t.Errorf("UserID = %+v, want the valid account %s", event.UserID, want.UserID)
	}
	assertPayload(t, "Data", event.Data, want.Payload)

	if want.RedactedPayload == nil {
		if event.RedactedData != nil {
			t.Errorf("RedactedData = %s, want nil for a payload without secrets", event.RedactedData)
		}
		return
	}
	assertPayload(t, "RedactedData", event.RedactedData, want.RedactedPayload)
}

func assertPayload(t testing.TB, field string, encoded json.RawMessage, want map[string]any) {
	t.Helper()

	decoded := map[string]any{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("%s %s is not a JSON object: %v", field, encoded, err)
	}
	if !reflect.DeepEqual(decoded, want) {
		t.Errorf("%s = %v, want exactly %v", field, decoded, want)
	}
}
