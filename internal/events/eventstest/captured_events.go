package eventstest

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

// CapturedEvents is an in-memory event log and broker: a dispatcher built from it appends each event to
// Recorded and Published instead of storing or sending it.
type CapturedEvents struct {
	Recorded  []events.NewEvent
	Published []events.NewEvent
}

// DispatchedBy is the one event a use case must have dispatched, and the account it must belong to.
type DispatchedBy struct {
	Label  events.Label
	UserID uuid.UUID
}

// Dispatcher returns a dispatcher that captures into c.
func (c *CapturedEvents) Dispatcher() events.Dispatcher {
	return events.Dispatcher{Recorder: c.Recorder(), Publisher: c.Publisher()}
}

// Recorder returns an event log recorder appending to Recorded.
func (c *CapturedEvents) Recorder() events.Recorder {
	return func(_ context.Context, event events.NewEvent) error {
		c.Recorded = append(c.Recorded, event)
		return nil
	}
}

// Publisher returns a broker publisher appending to Published.
func (c *CapturedEvents) Publisher() events.Publisher {
	return func(_ context.Context, event events.NewEvent) error {
		c.Published = append(c.Published, event)
		return nil
	}
}

// AssertNoneDispatched fails the test if any event was recorded or published.
func (c *CapturedEvents) AssertNoneDispatched(t testing.TB) {
	t.Helper()

	if len(c.Recorded) != 0 || len(c.Published) != 0 {
		t.Errorf("recorded/published events = %d/%d, want none", len(c.Recorded), len(c.Published))
	}
}

// OnlyDispatched fails the test unless exactly one event, labelled wantLabel, was recorded and published;
// it returns the published one.
func (c *CapturedEvents) OnlyDispatched(t testing.TB, wantLabel events.Label) events.NewEvent {
	t.Helper()

	if len(c.Recorded) != 1 || len(c.Published) != 1 {
		t.Fatalf("recorded/published events = %d/%d, want exactly one %q", len(c.Recorded), len(c.Published), wantLabel)
	}
	published := c.Published[0]
	if published.Label != wantLabel {
		t.Errorf("Label = %q, want %q", published.Label, wantLabel)
	}

	return published
}

// OnlyDispatchedBy is OnlyDispatched that also fails unless the event belongs to want.UserID.
func (c *CapturedEvents) OnlyDispatchedBy(t testing.TB, want DispatchedBy) events.NewEvent {
	t.Helper()

	published := c.OnlyDispatched(t, want.Label)
	if published.UserID != (uuid.NullUUID{UUID: want.UserID, Valid: true}) {
		t.Errorf("UserID = %+v, want %s", published.UserID, want.UserID)
	}

	return published
}

// OnlyRecorded fails the test unless exactly one event was recorded, and returns it.
func (c *CapturedEvents) OnlyRecorded(t testing.TB) events.NewEvent {
	t.Helper()

	if len(c.Recorded) != 1 {
		t.Fatalf("recorded events = %d, want 1", len(c.Recorded))
	}

	return c.Recorded[0]
}

// OnlyPublished fails the test unless exactly one event was published, and returns it.
func (c *CapturedEvents) OnlyPublished(t testing.TB) events.NewEvent {
	t.Helper()

	if len(c.Published) != 1 {
		t.Fatalf("published events = %d, want 1", len(c.Published))
	}

	return c.Published[0]
}

// DecodePayload returns the event's Data as a JSON object, failing the test when it is not one.
func DecodePayload(t testing.TB, event events.NewEvent) map[string]any {
	t.Helper()

	payload := map[string]any{}
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		t.Fatalf("event payload %s is not a JSON object: %v", event.Data, err)
	}

	return payload
}
