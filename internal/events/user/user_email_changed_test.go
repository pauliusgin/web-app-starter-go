package user

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

func TestNewUserEmailChangedCarriesBothAddresses(t *testing.T) {
	userID := uuid.New()

	changed, err := NewUserEmailChanged(NewUserEmailChangedParams{
		UserID:        userID,
		PreviousEmail: "accountant@example.com",
		Email:         "bookkeeper@example.com",
	})
	if err != nil {
		t.Fatalf("NewUserEmailChanged: %v", err)
	}

	if changed.Label != events.UserEmailChangedLabel {
		t.Errorf("Label = %q, want %q", changed.Label, events.UserEmailChangedLabel)
	}
	if changed.UserID != (uuid.NullUUID{UUID: userID, Valid: true}) {
		t.Errorf("UserID = %+v, want the valid account %s", changed.UserID, userID)
	}
	if changed.RedactedData != nil {
		t.Errorf("RedactedData = %s, want nil for a payload without secrets", changed.RedactedData)
	}

	payload := map[string]any{}
	if err := json.Unmarshal(changed.Data, &payload); err != nil {
		t.Fatalf("the encoded payload %s is not valid JSON: %v", changed.Data, err)
	}

	want := map[string]string{
		"userId":        userID.String(),
		"previousEmail": "accountant@example.com",
		"email":         "bookkeeper@example.com",
	}
	if len(payload) != len(want) {
		t.Errorf("payload = %s, want exactly %v", changed.Data, want)
	}
	for key, wantValue := range want {
		if payload[key] != wantValue {
			t.Errorf("payload[%q] = %v, want %q", key, payload[key], wantValue)
		}
	}
}
