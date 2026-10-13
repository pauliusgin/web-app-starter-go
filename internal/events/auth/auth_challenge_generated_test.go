package auth

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

const testChallengeToken = "9f1c4b7e2a0d48c6b3115ea77c0d5e93"

const testChallengeTarget = "accountant@example.com"

var testChallengeExpiresAt = time.Date(2026, time.September, 9, 18, 0, 0, 0, time.UTC)

func decodePayload(t *testing.T, rawPayload []byte) map[string]any {
	t.Helper()

	payload := map[string]any{}
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		t.Fatalf("the encoded payload %s is not valid JSON: %v", rawPayload, err)
	}

	return payload
}

func TestNewAuthChallengeGeneratedCarriesTheChallenge(t *testing.T) {
	userID := uuid.New()
	challengeID := uuid.New()

	generated, err := NewAuthChallengeGenerated(NewAuthChallengeParams{
		ChallengeID: challengeID,
		UserID:      userID,
		TargetType:  "email",
		Target:      testChallengeTarget,
		Purpose:     "email_verification",
		Token:       testChallengeToken,
		ExpiresAt:   testChallengeExpiresAt,
	})
	if err != nil {
		t.Fatalf("NewAuthChallengeGenerated: %v", err)
	}

	if generated.Label != events.AuthChallengeGeneratedLabel {
		t.Errorf("Label = %q, want %q", generated.Label, events.AuthChallengeGeneratedLabel)
	}
	if generated.UserID != (uuid.NullUUID{UUID: userID, Valid: true}) {
		t.Errorf("UserID = %+v, want the valid challenged account %s", generated.UserID, userID)
	}

	wantRedactedPayload := map[string]any{
		"challengeId": challengeID.String(),
		"userId":      userID.String(),
		"targetType":  "email",
		"target":      testChallengeTarget,
		"purpose":     "email_verification",
		"expiresAt":   testChallengeExpiresAt.Format(time.RFC3339Nano),
	}
	wantPayload := map[string]any{"token": testChallengeToken}
	for key, value := range wantRedactedPayload {
		wantPayload[key] = value
	}

	assertPayloadEquals(t, decodePayload(t, generated.Data), wantPayload)

	t.Run("redacted copy for the event log drops the token", func(t *testing.T) {
		if strings.Contains(string(generated.RedactedData), testChallengeToken) {
			t.Errorf("RedactedData = %s, want no trace of the plaintext token", generated.RedactedData)
		}
		assertPayloadEquals(t, decodePayload(t, generated.RedactedData), wantRedactedPayload)
	})
}

func assertPayloadEquals(t *testing.T, payload map[string]any, want map[string]any) {
	t.Helper()

	if len(payload) != len(want) {
		t.Errorf("payload = %v, want exactly the keys %v", payload, want)
	}
	for key, wantValue := range want {
		if payload[key] != wantValue {
			t.Errorf("%s = %v, want %v", key, payload[key], wantValue)
		}
	}
}
