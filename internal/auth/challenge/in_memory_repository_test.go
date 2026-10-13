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

type inMemoryChallengeStore struct {
	rows []Challenge

	saveFailure     error
	markUsedFailure error
}

func newInMemoryChallengeStore() *inMemoryChallengeStore {
	return &inMemoryChallengeStore{}
}

func (s *inMemoryChallengeStore) Save(
	_ context.Context,
	newChallenge NewChallenge,
) (Challenge, error) {
	if s.saveFailure != nil {
		return Challenge{}, s.saveFailure
	}

	storedChallenge := Challenge{
		ID:         newChallenge.ID,
		UserID:     newChallenge.UserID,
		TargetType: newChallenge.TargetType,
		Target:     newChallenge.Target,
		Purpose:    newChallenge.Purpose,
		TokenHash:  newChallenge.TokenHash,
		CreatedAt:  fixedCreatedAt,
		ExpiresAt:  newChallenge.ExpiresAt,
	}
	s.rows = append(s.rows, storedChallenge)

	return storedChallenge, nil
}

func (s *inMemoryChallengeStore) MarkUsed(_ context.Context, challengeID uuid.UUID) error {
	if s.markUsedFailure != nil {
		return s.markUsedFailure
	}

	for index, storedChallenge := range s.rows {
		if storedChallenge.ID != challengeID {
			continue
		}
		if storedChallenge.IsUsed() {
			return nil
		}

		s.rows[index].UsedAt = fixedUsedAt

		return nil
	}

	return nil
}

func (s *inMemoryChallengeStore) challengesForUser(userID uuid.UUID) []Challenge {
	var found []Challenge

	for _, storedChallenge := range s.rows {
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
