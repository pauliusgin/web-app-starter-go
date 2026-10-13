package challenge

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	"github.com/pauliusgin/web_app_starter-be/internal/events/auth"
)

// GenerateDependencies carries the collaborators generating a challenge needs.
type GenerateDependencies struct {
	WriteRepository WriteRepository
	EventDispatcher events.Dispatcher
}

// GenerateParams is the challenge being asked for.
type GenerateParams struct {
	UserID     uuid.UUID
	TargetType TargetType
	Target     string
	Purpose    Purpose
}

// Generated is what the caller receives, deliberately without the plaintext token.
type Generated struct {
	ID        uuid.UUID
	ExpiresAt time.Time
}

// Generate mints a challenge token, stores its hash, then records and publishes the event.
func Generate(
	ctx context.Context,
	d GenerateDependencies,
	params GenerateParams,
) (Generated, error) {

	if err := ensureChallengeRequestComplete(params); err != nil {
		return Generated{}, err
	}

	minted := token.Generate()

	expiresAt := time.Now().Add(Lifetime)

	storedChallenge, err := d.WriteRepository.Save(ctx, NewChallenge{
		ID:         uuid.New(),
		UserID:     params.UserID,
		TargetType: params.TargetType,
		Target:     params.Target,
		Purpose:    params.Purpose,
		TokenHash:  minted.Hash,
		ExpiresAt:  expiresAt,
	})
	if err != nil {
		return Generated{}, err
	}

	generated := Generated{ID: storedChallenge.ID, ExpiresAt: storedChallenge.ExpiresAt}

	auth.DispatchAuthChallengeGenerated(ctx, d.EventDispatcher, auth.NewAuthChallengeParams{
		ChallengeID: storedChallenge.ID,
		UserID:      storedChallenge.UserID,
		TargetType:  string(storedChallenge.TargetType),
		Target:      storedChallenge.Target,
		Purpose:     string(storedChallenge.Purpose),
		Token:       minted.Plaintext,
		ExpiresAt:   storedChallenge.ExpiresAt,
	})

	return generated, nil
}

func ensureChallengeRequestComplete(params GenerateParams) error {
	if params.Target == "" {
		return ErrTargetMissing{}
	}
	if params.TargetType == "" {
		return ErrTargetTypeMissing{}
	}
	if params.Purpose == "" {
		return ErrPurposeMissing{}
	}

	return nil
}
