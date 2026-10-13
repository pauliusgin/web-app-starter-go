package user

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

const testEmail = "accountant@example.com"

type assertAccountEventParams struct {
	event     events.NewEvent
	wantLabel events.Label
	userID    uuid.UUID
}

func assertAccountEvent(t *testing.T, params assertAccountEventParams) {
	t.Helper()

	if params.event.Label != params.wantLabel {
		t.Errorf("Label = %q, want %q", params.event.Label, params.wantLabel)
	}
	if params.event.UserID != (uuid.NullUUID{UUID: params.userID, Valid: true}) {
		t.Errorf("UserID = %+v, want the valid account %s", params.event.UserID, params.userID)
	}
	if params.event.RedactedData != nil {
		t.Errorf("RedactedData = %s, want nil for a payload without secrets", params.event.RedactedData)
	}

	payload := map[string]any{}
	if err := json.Unmarshal(params.event.Data, &payload); err != nil {
		t.Fatalf("the encoded payload %s is not valid JSON: %v", params.event.Data, err)
	}
	if len(payload) != 2 {
		t.Errorf("payload = %s, want only userId and email", params.event.Data)
	}
	if payload["userId"] != params.userID.String() {
		t.Errorf("userId = %v, want %s", payload["userId"], params.userID)
	}
	if payload["email"] != testEmail {
		t.Errorf("email = %v, want %q", payload["email"], testEmail)
	}
}

func TestNewUserSignedUpCarriesTheAccount(t *testing.T) {
	userID := uuid.New()

	signedUp, err := NewUserSignedUp(NewUserSignedUpParams{UserID: userID, Email: testEmail})
	if err != nil {
		t.Fatalf("NewUserSignedUp: %v", err)
	}

	assertAccountEvent(t, assertAccountEventParams{
		event:     signedUp,
		wantLabel: events.UserSignedUpLabel,
		userID:    userID,
	})
}
