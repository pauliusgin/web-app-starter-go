// Package usecases holds the profile use cases: recording and changing a user's personal details.
package usecases

import (
	"context"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	profileevents "github.com/pauliusgin/web_app_starter-be/internal/events/profile"
)

// CreateProfileParams is the authenticated user and the names their profile starts with.
type CreateProfileParams struct {
	UserID    uuid.UUID
	FirstName string
	LastName  string
}

// CreateProfileResult is the profile the user just recorded.
type CreateProfileResult struct {
	Profile models.Profile
}

// CreateProfileDependencies carries the collaborators creating a profile needs.
type CreateProfileDependencies struct {
	ProfileRepository repositories.ProfileRepository
	EventDispatcher   events.Dispatcher
}

// CreateProfile records the user's profile once, keyed by their user ID, then announces it.
func CreateProfile(ctx context.Context, d CreateProfileDependencies, params CreateProfileParams) (CreateProfileResult, error) {
	profile, err := d.ProfileRepository.SaveProfile(ctx, models.NewProfile{
		ID:        params.UserID,
		FirstName: params.FirstName,
		LastName:  params.LastName,
	})
	if err != nil {
		return CreateProfileResult{}, err
	}

	profileevents.DispatchProfileCreated(ctx, d.EventDispatcher, profileevents.NewProfileCreatedParams{
		UserID:    profile.ID,
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
	})

	return CreateProfileResult{Profile: profile}, nil
}
