package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/eventstest"
)

func TestAuthChallengeGeneratedSendsTheTokenToSubscribersButNotToTheLog(t *testing.T) {
	userID := uuid.MustParse("1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15")
	challengeID := uuid.MustParse("6f1c2d3e-4b5a-4c6d-8e7f-9a0b1c2d3e4f")
	expiresAt := time.Date(2026, time.September, 9, 18, 0, 0, 0, time.UTC)

	generated, err := NewAuthChallengeGenerated(NewAuthChallengeGeneratedParams{
		ChallengeID: challengeID,
		UserID:      userID,
		TargetType:  "email",
		Target:      "accountant@example.com",
		Purpose:     "email_verification",
		Token:       "9f1c4b7e2a0d48c6b3115ea77c0d5e93",
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		t.Fatalf("NewAuthChallengeGenerated: %v", err)
	}

	redacted := map[string]any{
		"challengeId": challengeID.String(),
		"userId":      userID.String(),
		"targetType":  "email",
		"target":      "accountant@example.com",
		"purpose":     "email_verification",
		"expiresAt":   expiresAt.Format(time.RFC3339Nano),
	}
	withToken := map[string]any{"token": "9f1c4b7e2a0d48c6b3115ea77c0d5e93"}
	for key, value := range redacted {
		withToken[key] = value
	}

	eventstest.AssertEvent(t, generated, eventstest.WantEvent{
		Label:           events.AuthChallengeGeneratedLabel,
		UserID:          userID,
		Payload:         withToken,
		RedactedPayload: redacted,
	})
}
