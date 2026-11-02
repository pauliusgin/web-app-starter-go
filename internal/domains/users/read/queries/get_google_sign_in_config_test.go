package queries

import (
	"context"
	"testing"
)

func TestGetGoogleSignInConfigReturnsTheConfiguredClientID(t *testing.T) {
	const clientID = "1234567890-abc123def456.apps.googleusercontent.com"

	result, err := GetGoogleSignInConfig(context.Background(), GetGoogleSignInConfigDependencies{
		GoogleClientID: clientID,
	})
	if err != nil {
		t.Fatalf("GetGoogleSignInConfig: %v", err)
	}

	if result.ClientID != clientID {
		t.Errorf("ClientID = %q, want %q", result.ClientID, clientID)
	}
}
