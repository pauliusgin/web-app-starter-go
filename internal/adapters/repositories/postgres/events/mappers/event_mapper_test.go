package mappers

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

func TestEventMapperMapsEveryColumn(t *testing.T) {
	row := eventLogRow(uuid.NullUUID{
		UUID:  uuid.MustParse("1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15"),
		Valid: true,
	})

	mapped := EventMapper(row)

	if mapped.ID != row.ID {
		t.Errorf("ID = %s, want %s", mapped.ID, row.ID)
	}
	if mapped.UserID != row.UserID {
		t.Errorf("UserID = %v, want %v", mapped.UserID, row.UserID)
	}
	if mapped.Label != events.UserSignedUpLabel {
		t.Errorf("Label = %q, want %q", mapped.Label, events.UserSignedUpLabel)
	}
	if !bytes.Equal(mapped.Data, row.EventData) {
		t.Errorf("Data = %s, want %s", mapped.Data, row.EventData)
	}
	if !mapped.OccurredAt.Equal(row.OccurredAt) {
		t.Errorf("OccurredAt = %v, want %v", mapped.OccurredAt, row.OccurredAt)
	}
}

func TestEventMapperKeepsNullUserIDNull(t *testing.T) {
	mapped := EventMapper(eventLogRow(uuid.NullUUID{}))

	if mapped.UserID.Valid {
		t.Errorf("UserID = %v, want null for an event dispatched without a user", mapped.UserID)
	}
}

func eventLogRow(userID uuid.NullUUID) sqlc.EventLog {
	return sqlc.EventLog{
		ID:         uuid.MustParse("7b2f6f7a-2a1e-4c3d-9f10-6d4c8b1e5a20"),
		UserID:     userID,
		EventLabel: string(events.UserSignedUpLabel),
		EventData:  json.RawMessage(`{"email":"jane@example.com"}`),
		OccurredAt: time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC),
	}
}
