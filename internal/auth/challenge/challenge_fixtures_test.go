package challenge

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

var fixedCreatedAt = time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)

var fixedUsedAt = time.Date(2026, time.September, 9, 13, 0, 0, 0, time.UTC)

func challengesForUser(repository *InMemoryChallengeRepository, userID uuid.UUID) []Challenge {
	var found []Challenge

	for _, storedChallenge := range repository.Challenges {
		if storedChallenge.UserID == userID {
			found = append(found, storedChallenge)
		}
	}

	return found
}

func tokenFromEventData(t *testing.T, event events.NewEvent) string {
	t.Helper()

	payload := struct {
		Token string `json:"token"`
	}{}

	if err := json.Unmarshal(event.Data, &payload); err != nil {
		t.Fatalf("unmarshalling the event data: %v", err)
	}

	return payload.Token
}
