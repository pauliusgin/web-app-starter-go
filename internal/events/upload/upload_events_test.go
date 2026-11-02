package upload

import (
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/eventstest"
)

func TestUploadEventsCarryThePayloadSubscribersRead(t *testing.T) {
	userID := uuid.MustParse("1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15")
	uploadID := uuid.MustParse("6f1c2d3e-4b5a-4c6d-8e7f-9a0b1c2d3e4f")

	cases := []struct {
		name  string
		build func() (events.NewEvent, error)
		want  eventstest.WantEvent
	}{
		{
			name: "created",
			build: func() (events.NewEvent, error) {
				return NewUploadCreated(NewUploadCreatedParams{UploadID: uploadID, UserID: userID, Purpose: "company_logo", Kind: "image"})
			},
			want: eventstest.WantEvent{Label: events.UploadCreatedLabel, UserID: userID, Payload: map[string]any{
				"uploadId": uploadID.String(), "userId": userID.String(), "purpose": "company_logo", "kind": "image",
			}},
		},
		{
			name: "deleted",
			build: func() (events.NewEvent, error) {
				return NewUploadDeleted(NewUploadDeletedParams{UploadID: uploadID, UserID: userID, Purpose: "invoice_picture_before"})
			},
			want: eventstest.WantEvent{Label: events.UploadDeletedLabel, UserID: userID, Payload: map[string]any{
				"uploadId": uploadID.String(), "userId": userID.String(), "purpose": "invoice_picture_before",
			}},
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
