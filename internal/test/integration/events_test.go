package integration

import (
	"strings"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

func TestEventLogStoresTheRedactedPayloadWhileSubscribersGetTheSecret(t *testing.T) {
	world := newIntegrationWorld(t)
	signedUp := world.signUp(t, "accountant@example.com")
	token := world.latestChallengeToken(t, "accountant@example.com")

	var loggedPayload string
	err := world.database.QueryRowContext(world.ctx,
		`SELECT event_data::text FROM event_log WHERE event_label = $1 AND user_id = $2`,
		string(events.AuthChallengeGeneratedLabel), signedUp.User.ID).Scan(&loggedPayload)
	if err != nil {
		t.Fatalf("reading the logged challenge event: %v", err)
	}

	if strings.Contains(loggedPayload, token) {
		t.Errorf("event_log payload = %s, want the plaintext token left out", loggedPayload)
	}
	if !strings.Contains(loggedPayload, "accountant@example.com") {
		t.Errorf("event_log payload = %s, want the non-secret fields kept", loggedPayload)
	}
}
