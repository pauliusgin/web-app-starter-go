package profile

import (
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/eventstest"
)

func TestProfileEventsKeepTheNamesOutOfTheEventLog(t *testing.T) {
	userID := uuid.MustParse("1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15")
	named := map[string]any{"userId": userID.String(), "firstName": "Ada", "lastName": "Lovelace"}
	redacted := map[string]any{"userId": userID.String()}

	cases := []struct {
		name  string
		build func() (events.NewEvent, error)
		want  eventstest.WantEvent
	}{
		{
			name: "created",
			build: func() (events.NewEvent, error) {
				return NewProfileCreated(NewProfileCreatedParams{UserID: userID, FirstName: "Ada", LastName: "Lovelace"})
			},
			want: eventstest.WantEvent{Label: events.ProfileCreatedLabel, UserID: userID, Payload: named, RedactedPayload: redacted},
		},
		{
			name: "updated",
			build: func() (events.NewEvent, error) {
				return NewProfileUpdated(NewProfileUpdatedParams{UserID: userID, FirstName: "Ada", LastName: "Lovelace"})
			},
			want: eventstest.WantEvent{Label: events.ProfileUpdatedLabel, UserID: userID, Payload: named, RedactedPayload: redacted},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			built, err := testCase.build()
			if err != nil {
				t.Fatalf("building the event: %v", err)
			}

			eventstest.AssertEvent(t, built, testCase.want)
		})
	}
}
