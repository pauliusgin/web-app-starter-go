package user

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

func TestNewUserDeletedCarriesOnlyTheUserID(t *testing.T) {
	userID := uuid.New()

	deleted, err := NewUserDeleted(NewUserDeletedParams{UserID: userID})
	if err != nil {
		t.Fatalf("NewUserDeleted: %v", err)
	}

	if deleted.Label != events.UserDeletedLabel {
		t.Errorf("Label = %q, want %q", deleted.Label, events.UserDeletedLabel)
	}
	if deleted.UserID != (uuid.NullUUID{UUID: userID, Valid: true}) {
		t.Errorf("UserID = %+v, want the valid account %s", deleted.UserID, userID)
	}
	if deleted.RedactedData != nil {
		t.Errorf("RedactedData = %s, want nil for a payload without secrets", deleted.RedactedData)
	}

	payload := map[string]any{}
	if err := json.Unmarshal(deleted.Data, &payload); err != nil {
		t.Fatalf("the encoded payload %s is not valid JSON: %v", deleted.Data, err)
	}
	if len(payload) != 1 || payload["userId"] != userID.String() {
		t.Errorf("payload = %s, want only userId %s", deleted.Data, userID)
	}
}
