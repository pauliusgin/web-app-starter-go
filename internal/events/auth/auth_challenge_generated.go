// Package auth defines the events dispatched while authenticating an account.
package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

// AuthChallengeGeneratedData is the broker payload, carrying the plaintext token for the link.
type AuthChallengeGeneratedData struct {
	ChallengeID uuid.UUID `json:"challengeId"`
	UserID      uuid.UUID `json:"userId"`
	TargetType  string    `json:"targetType"`
	Target      string    `json:"target"`
	Purpose     string    `json:"purpose"`
	Token       string    `json:"token"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

// authChallengeGeneratedRedactedData is the broker payload minus the token: the event log's copy.
type authChallengeGeneratedRedactedData struct {
	ChallengeID uuid.UUID `json:"challengeId"`
	UserID      uuid.UUID `json:"userId"`
	TargetType  string    `json:"targetType"`
	Target      string    `json:"target"`
	Purpose     string    `json:"purpose"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

// NewAuthChallengeParams is the challenge an auth_challenge.generated event carries.
type NewAuthChallengeParams struct {
	ChallengeID uuid.UUID
	UserID      uuid.UUID
	TargetType  string
	Target      string
	Purpose     string
	Token       string
	ExpiresAt   time.Time
}

// NewAuthChallengeGenerated builds the event announcing a freshly minted verification challenge.
func NewAuthChallengeGenerated(params NewAuthChallengeParams) (events.NewEvent, error) {
	return events.New(events.NewEventParams{
		Label:  events.AuthChallengeGeneratedLabel,
		UserID: uuid.NullUUID{UUID: params.UserID, Valid: true},
		Data: AuthChallengeGeneratedData{
			ChallengeID: params.ChallengeID,
			UserID:      params.UserID,
			TargetType:  params.TargetType,
			Target:      params.Target,
			Purpose:     params.Purpose,
			Token:       params.Token,
			ExpiresAt:   params.ExpiresAt,
		},
		RedactedData: authChallengeGeneratedRedactedData{
			ChallengeID: params.ChallengeID,
			UserID:      params.UserID,
			TargetType:  params.TargetType,
			Target:      params.Target,
			Purpose:     params.Purpose,
			ExpiresAt:   params.ExpiresAt,
		},
	})
}

// DispatchAuthChallengeGenerated builds, records and publishes the event announcing a minted challenge.
func DispatchAuthChallengeGenerated(ctx context.Context, dispatcher events.Dispatcher, params NewAuthChallengeParams) {
	dispatcher.Dispatch(ctx, func() (events.NewEvent, error) {
		return NewAuthChallengeGenerated(params)
	})
}
