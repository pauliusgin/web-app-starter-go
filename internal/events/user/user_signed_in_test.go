package user

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

func TestNewUserSignedInCarriesTheAccount(t *testing.T) {
	userID := uuid.New()

	signedIn, err := NewUserSignedIn(NewUserSignedInParams{UserID: userID, Email: testEmail})
	if err != nil {
		t.Fatalf("NewUserSignedIn: %v", err)
	}

	assertAccountEvent(t, assertAccountEventParams{
		event:     signedIn,
		wantLabel: events.UserSignedInLabel,
		userID:    userID,
	})
}
