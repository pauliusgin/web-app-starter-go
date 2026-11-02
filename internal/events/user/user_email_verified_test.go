package user

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

func TestEmailVerifiedCarriesTheAccount(t *testing.T) {
	userID := uuid.New()

	verified, err := NewUserEmailVerified(NewUserEmailVerifiedParams{UserID: userID, Email: testEmail})
	if err != nil {
		t.Fatalf("EmailVerified: %v", err)
	}

	assertAccountEvent(t, assertAccountEventParams{
		event:     verified,
		wantLabel: events.UserEmailVerifiedLabel,
		userID:    userID,
	})
}
