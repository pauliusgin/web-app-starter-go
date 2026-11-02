package queries

import "context"

// GetGoogleSignInConfigResult is what a client needs to render the Sign in with Google button.
type GetGoogleSignInConfigResult struct {
	ClientID string
}

// GetGoogleSignInConfigDependencies carries the google OAuth client ID this service accepts ID tokens for.
type GetGoogleSignInConfigDependencies struct {
	GoogleClientID string
}

// GetGoogleSignInConfig returns the google OAuth client ID, so the server stays its only source of truth.
func GetGoogleSignInConfig(_ context.Context, d GetGoogleSignInConfigDependencies) (GetGoogleSignInConfigResult, error) {
	return GetGoogleSignInConfigResult{ClientID: d.GoogleClientID}, nil
}
