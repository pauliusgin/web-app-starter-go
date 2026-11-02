// Package usecases holds the profile use cases: recording and changing a user's personal details.
package usecases

import (
	"context"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/repositories"
	uploadmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	profileevents "github.com/pauliusgin/web_app_starter-be/internal/events/profile"
)

// AttachUpload is the upload claim that keeps a profile's picture from being cleaned up.
type AttachUpload func(ctx context.Context, params uploadmodels.AttachUploadParams) (uploadmodels.Upload, error)

// CreateProfileParams is the authenticated user and the names and optional picture their profile starts with;
// a uuid.Nil PictureUploadID is no picture.
type CreateProfileParams struct {
	UserID          uuid.UUID
	FirstName       string
	LastName        string
	PictureUploadID uuid.UUID
}

// CreateProfileResult is the profile the user just recorded.
type CreateProfileResult struct {
	Profile models.Profile
}

// CreateProfileDependencies carries the profile store, the dispatcher announcing the creation and the picture
// attach call.
type CreateProfileDependencies struct {
	ProfileWriteRepository repositories.ProfileWriteRepository
	EventDispatcher        events.Dispatcher
	AttachUpload           AttachUpload
}

// CreateProfile records the user's profile once, keyed by their user ID, claiming its picture upload first,
// then announces it.
func CreateProfile(ctx context.Context, d CreateProfileDependencies, params CreateProfileParams) (CreateProfileResult, error) {
	err := attachProfilePicture(ctx, attachProfilePictureParams{
		AttachUpload:    d.AttachUpload,
		UserID:          params.UserID,
		PictureUploadID: params.PictureUploadID,
	})
	if err != nil {
		return CreateProfileResult{}, err
	}

	profile, err := d.ProfileWriteRepository.SaveProfile(ctx, models.NewProfile{
		ID:              params.UserID,
		FirstName:       params.FirstName,
		LastName:        params.LastName,
		PictureUploadID: params.PictureUploadID,
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

type attachProfilePictureParams struct {
	AttachUpload    AttachUpload
	UserID          uuid.UUID
	PictureUploadID uuid.UUID
}

func attachProfilePicture(ctx context.Context, params attachProfilePictureParams) error {
	if params.PictureUploadID == uuid.Nil {
		return nil
	}

	_, err := params.AttachUpload(ctx, uploadmodels.AttachUploadParams{
		ID:      params.PictureUploadID,
		UserID:  params.UserID,
		Purpose: uploadmodels.PurposeProfilePicture,
	})

	return err
}
