package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/models"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

var testUserID = uuid.MustParse("1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15")

var storedAt = time.Date(2026, time.October, 12, 9, 0, 0, 0, time.UTC)

var errDatabaseDown = errors.New("database unreachable")

func storedProfile() models.Profile {
	return models.Profile{
		ID:        testUserID,
		FirstName: "Ada",
		LastName:  "Lovelace",
		CreatedAt: storedAt,
		UpdatedAt: storedAt,
	}
}

type capturedEvents struct {
	recorded  []events.NewEvent
	published []events.NewEvent
}

func (c *capturedEvents) dispatcher() events.Dispatcher {
	return events.Dispatcher{
		Recorder: func(_ context.Context, event events.NewEvent) error {
			c.recorded = append(c.recorded, event)
			return nil
		},
		Publisher: func(_ context.Context, event events.NewEvent) error {
			c.published = append(c.published, event)
			return nil
		},
	}
}
