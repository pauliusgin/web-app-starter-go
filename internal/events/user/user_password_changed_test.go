package user

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

func TestNewUserPasswordChangedCarriesOnlyTheUserID(t *testing.T) {
	userID := uuid.New()

	changed, err := NewUserPasswordChanged(NewUserPasswordChangedParams{UserID: userID})
	if err != nil {
		t.Fatalf("NewUserPasswordChanged: %v", err)
	}

	if changed.Label != events.UserPasswordChangedLabel {
		t.Errorf("Label = %q, want %q", changed.Label, events.UserPasswordChangedLabel)
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
	if len(payload) != 1 || payload["userId"] != userID.String() {
		t.Errorf("payload = %s, want only userId %s", changed.Data, userID)
	}
}
