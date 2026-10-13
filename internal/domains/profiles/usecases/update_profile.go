package usecases

import (
	"context"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	profileevents "github.com/pauliusgin/web_app_starter-be/internal/events/profile"
)

// UpdateProfileParams is the authenticated user and the full set of names replacing their current ones.
type UpdateProfileParams struct {
	UserID    uuid.UUID
	FirstName string
	LastName  string
}

// UpdateProfileResult is the profile with its replaced names.
type UpdateProfileResult struct {
	Profile models.Profile
}

// UpdateProfileDependencies carries the collaborators updating a profile needs.
type UpdateProfileDependencies struct {
	ProfileRepository repositories.ProfileRepository
	EventDispatcher   events.Dispatcher
}

// UpdateProfile replaces the names on the user's existing profile, then announces it.
func UpdateProfile(ctx context.Context, d UpdateProfileDependencies, params UpdateProfileParams) (UpdateProfileResult, error) {
	profile, err := d.ProfileRepository.UpdateProfile(ctx, models.UpdateProfileParams{
		ID:        params.UserID,
		FirstName: params.FirstName,
		LastName:  params.LastName,
	})
	if err != nil {
		return UpdateProfileResult{}, err
	}

	profileevents.DispatchProfileUpdated(ctx, d.EventDispatcher, profileevents.NewProfileUpdatedParams{
		UserID:    profile.ID,
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
	})

	return UpdateProfileResult{Profile: profile}, nil
}
