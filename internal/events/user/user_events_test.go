package user

import (
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/eventstest"
)

func TestUserEventsCarryThePayloadSubscribersRead(t *testing.T) {
	userID := uuid.MustParse("1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15")
	account := map[string]any{"userId": userID.String(), "email": "accountant@example.com"}

	cases := []struct {
		name  string
		build func() (events.NewEvent, error)
		want  eventstest.WantEvent
	}{
		{
			name: "signed up",
			build: func() (events.NewEvent, error) {
				return NewUserSignedUp(NewUserSignedUpParams{UserID: userID, Email: "accountant@example.com"})
			},
			want: eventstest.WantEvent{Label: events.UserSignedUpLabel, UserID: userID, Payload: account},
		},
		{
			name: "signed in",
			build: func() (events.NewEvent, error) {
				return NewUserSignedIn(NewUserSignedInParams{UserID: userID, Email: "accountant@example.com"})
			},
			want: eventstest.WantEvent{Label: events.UserSignedInLabel, UserID: userID, Payload: account},
		},
		{
			name: "email verified",
			build: func() (events.NewEvent, error) {
				return NewUserEmailVerified(NewUserEmailVerifiedParams{UserID: userID, Email: "accountant@example.com"})
			},
			want: eventstest.WantEvent{Label: events.UserEmailVerifiedLabel, UserID: userID, Payload: account},
		},
		{
			name: "email changed",
			build: func() (events.NewEvent, error) {
				return NewUserEmailChanged(NewUserEmailChangedParams{
					UserID:        userID,
					PreviousEmail: "accountant@example.com",
					Email:         "bookkeeper@example.com",
				})
			},
			want: eventstest.WantEvent{Label: events.UserEmailChangedLabel, UserID: userID, Payload: map[string]any{
				"userId":        userID.String(),
				"previousEmail": "accountant@example.com",
				"email":         "bookkeeper@example.com",
			}},
		},
		{
			name: "password changed",
			build: func() (events.NewEvent, error) {
				return NewUserPasswordChanged(NewUserPasswordChangedParams{UserID: userID})
			},
			want: eventstest.WantEvent{
				Label:   events.UserPasswordChangedLabel,
				UserID:  userID,
				Payload: map[string]any{"userId": userID.String()},
			},
		},
		{
			name: "password reset",
			build: func() (events.NewEvent, error) {
				return NewUserPasswordReset(NewUserPasswordResetParams{UserID: userID})
			},
			want: eventstest.WantEvent{
				Label:   events.UserPasswordResetLabel,
				UserID:  userID,
				Payload: map[string]any{"userId": userID.String()},
			},
		},
		{
			name: "password added",
			build: func() (events.NewEvent, error) {
				return NewUserPasswordAdded(NewUserPasswordAddedParams{UserID: userID})
			},
			want: eventstest.WantEvent{
				Label:   events.UserPasswordAddedLabel,
				UserID:  userID,
				Payload: map[string]any{"userId": userID.String()},
			},
		},
		{
			name: "google account added",
			build: func() (events.NewEvent, error) {
				return NewUserGoogleAccountAdded(NewUserGoogleAccountAddedParams{UserID: userID})
			},
			want: eventstest.WantEvent{
				Label:   events.UserGoogleAccountAddedLabel,
				UserID:  userID,
				Payload: map[string]any{"userId": userID.String()},
			},
		},
		{
			name:  "deleted",
			build: func() (events.NewEvent, error) { return NewUserDeleted(NewUserDeletedParams{UserID: userID}) },
			want: eventstest.WantEvent{
				Label:   events.UserDeletedLabel,
				UserID:  userID,
				Payload: map[string]any{"userId": userID.String()},
			},
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
