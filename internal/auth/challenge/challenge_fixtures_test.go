package challenge

import (
	"context"
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

type capturedEvents struct {
	recorded  []events.NewEvent
	published []events.NewEvent
}

func (c *capturedEvents) recorder() events.Recorder {
	return func(_ context.Context, event events.NewEvent) error {
		c.recorded = append(c.recorded, event)
		return nil
	}
}

func (c *capturedEvents) publisher() events.Publisher {
	return func(_ context.Context, event events.NewEvent) error {
		c.published = append(c.published, event)
		return nil
	}
}

func (c *capturedEvents) onlyRecorded(t *testing.T) events.NewEvent {
	t.Helper()

	if len(c.recorded) != 1 {
		t.Fatalf("recorded events = %d, want 1", len(c.recorded))
	}

	return c.recorded[0]
}

func (c *capturedEvents) onlyPublished(t *testing.T) events.NewEvent {
	t.Helper()

	if len(c.published) != 1 {
		t.Fatalf("published events = %d, want 1", len(c.published))
	}

	return c.published[0]
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
