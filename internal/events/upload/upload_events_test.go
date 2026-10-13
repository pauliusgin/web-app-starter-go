package upload

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

var testUploadID = uuid.MustParse("6f1c2d3e-4b5a-4c6d-8e7f-9a0b1c2d3e4f")

var testUserID = uuid.MustParse("1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15")

func decodePayload(t *testing.T, event events.NewEvent) map[string]any {
	t.Helper()

	payload := map[string]any{}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		t.Fatalf("the encoded payload %s is not valid JSON: %v", event.Data, err)
	}

	return payload
}

func assertLabelAndUser(t *testing.T, event events.NewEvent, wantLabel events.Label) {
	t.Helper()

	if event.Label != wantLabel {
		t.Errorf("Label = %q, want %q", event.Label, wantLabel)
	}
	if event.UserID != (uuid.NullUUID{UUID: testUserID, Valid: true}) {
		t.Errorf("UserID = %+v, want the valid account %s", event.UserID, testUserID)
	}
	if event.RedactedData != nil {
		t.Errorf("RedactedData = %s, want nil for a payload without secrets", event.RedactedData)
	}
}

func TestNewUploadCreatedCarriesTheUploadOwnerPurposeAndKind(t *testing.T) {
	created, err := NewUploadCreated(NewUploadCreatedParams{
		UploadID: testUploadID,
		UserID:   testUserID,
		Purpose:  "profile_picture",
		Kind:     "image",
	})
	if err != nil {
		t.Fatalf("NewUploadCreated: %v", err)
	}

	assertLabelAndUser(t, created, events.UploadCreatedLabel)

	payload := decodePayload(t, created)
	want := map[string]any{
		"uploadId": testUploadID.String(),
		"userId":   testUserID.String(),
		"purpose":  "profile_picture",
		"kind":     "image",
	}
	if len(payload) != len(want) {
		t.Fatalf("payload = %s, want exactly %v", created.Data, want)
	}
	for key, wantValue := range want {
		if payload[key] != wantValue {
			t.Errorf("payload[%q] = %v, want %v", key, payload[key], wantValue)
		}
	}
}

func TestNewUploadDeletedCarriesTheUploadOwnerAndPurpose(t *testing.T) {
	deleted, err := NewUploadDeleted(NewUploadDeletedParams{
		UploadID: testUploadID,
		UserID:   testUserID,
		Purpose:  "document_attachment",
	})
	if err != nil {
		t.Fatalf("NewUploadDeleted: %v", err)
	}

	assertLabelAndUser(t, deleted, events.UploadDeletedLabel)

	payload := decodePayload(t, deleted)
	want := map[string]any{
		"uploadId": testUploadID.String(),
		"userId":   testUserID.String(),
		"purpose":  "document_attachment",
	}
	if len(payload) != len(want) {
		t.Fatalf("payload = %s, want exactly %v", deleted.Data, want)
	}
	for key, wantValue := range want {
		if payload[key] != wantValue {
			t.Errorf("payload[%q] = %v, want %v", key, payload[key], wantValue)
		}
	}
}
