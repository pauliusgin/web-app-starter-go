package usecases

import (
	"context"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	profileevents "github.com/pauliusgin/web_app_starter-be/internal/events/profile"
)

// UpdateProfileParams is the authenticated user and the full set of names and picture replacing their current
// ones; a uuid.Nil PictureUploadID clears the picture.
type UpdateProfileParams struct {
	UserID          uuid.UUID
	FirstName       string
	LastName        string
	PictureUploadID uuid.UUID
}

// UpdateProfileResult is the profile with its replaced names.
type UpdateProfileResult struct {
	Profile models.Profile
}

// UpdateProfileDependencies carries the profile store, the dispatcher announcing the update and the picture
// attach call.
type UpdateProfileDependencies struct {
	ProfileWriteRepository repositories.ProfileWriteRepository
	EventDispatcher        events.Dispatcher
	AttachUpload           AttachUpload
}

// UpdateProfile replaces the names and picture on the user's existing profile, claiming a new picture upload
// first, then announces it. A replaced picture is left to the orphaned upload cleanup.
func UpdateProfile(ctx context.Context, d UpdateProfileDependencies, params UpdateProfileParams) (UpdateProfileResult, error) {
	err := attachProfilePicture(ctx, attachProfilePictureParams{
		AttachUpload:    d.AttachUpload,
		UserID:          params.UserID,
		PictureUploadID: params.PictureUploadID,
	})
	if err != nil {
		return UpdateProfileResult{}, err
	}

	profile, err := d.ProfileWriteRepository.UpdateProfile(ctx, models.UpdateProfileParams{
		ID:              params.UserID,
		FirstName:       params.FirstName,
		LastName:        params.LastName,
		PictureUploadID: params.PictureUploadID,
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
