package profile

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

var testUserID = uuid.MustParse("1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15")

// wantProfileEvent is what a profile event must carry: names for the broker, none for the event log.
type wantProfileEvent struct {
	label           events.Label
	payload         map[string]any
	redactedPayload map[string]any
}

func assertProfileEvent(t *testing.T, event events.NewEvent, want wantProfileEvent) {
	t.Helper()

	if event.Label != want.label {
		t.Errorf("Label = %q, want %q", event.Label, want.label)
	}
	if event.UserID != (uuid.NullUUID{UUID: testUserID, Valid: true}) {
		t.Errorf("UserID = %+v, want the valid account %s", event.UserID, testUserID)
	}
	assertPayloadEquals(t, event.Data, want.payload)
	assertPayloadEquals(t, event.RedactedData, want.redactedPayload)
}

func assertPayloadEquals(t *testing.T, encoded []byte, want map[string]any) {
	t.Helper()

	payload := map[string]any{}
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("the encoded payload %s is not valid JSON: %v", encoded, err)
	}
	if len(payload) != len(want) {
		t.Errorf("payload = %s, want exactly the keys of %v", encoded, want)
	}
	for key, wantValue := range want {
		if payload[key] != wantValue {
			t.Errorf("payload[%q] = %v, want %v", key, payload[key], wantValue)
		}
	}
}

func TestNewProfileCreatedSendsNamesToBrokerButNotToLog(t *testing.T) {
	created, err := NewProfileCreated(NewProfileCreatedParams{
		UserID:    testUserID,
		FirstName: "Ada",
		LastName:  "Lovelace",
	})
	if err != nil {
		t.Fatalf("NewProfileCreated: %v", err)
	}

	assertProfileEvent(t, created, wantProfileEvent{
		label: events.ProfileCreatedLabel,
		payload: map[string]any{
			"userId":    testUserID.String(),
			"firstName": "Ada",
			"lastName":  "Lovelace",
		},
		redactedPayload: map[string]any{"userId": testUserID.String()},
	})
}

func TestNewProfileUpdatedSendsNamesToBrokerButNotToLog(t *testing.T) {
	updated, err := NewProfileUpdated(NewProfileUpdatedParams{
		UserID:    testUserID,
		FirstName: "Augusta",
		LastName:  "King",
	})
	if err != nil {
		t.Fatalf("NewProfileUpdated: %v", err)
	}

	assertProfileEvent(t, updated, wantProfileEvent{
		label: events.ProfileUpdatedLabel,
		payload: map[string]any{
			"userId":    testUserID.String(),
			"firstName": "Augusta",
			"lastName":  "King",
		},
		redactedPayload: map[string]any{"userId": testUserID.String()},
	})
}
