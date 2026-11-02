package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/errors"
	readmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/read/models"
	read "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/read/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/models"
)

var _ ProfileWriteRepository = (*InMemoryProfileRepository)(nil)
var _ read.ProfileReadRepository = (*InMemoryProfileRepository)(nil)

// InMemoryProfileRepository is both profile ports over one map of profiles keyed by user ID.
type InMemoryProfileRepository struct {
	Profiles map[uuid.UUID]models.Profile

	// SaveFailure, UpdateFailure and FindFailure, when set, are returned instead of doing the work.
	SaveFailure   error
	UpdateFailure error
	FindFailure   error
}

// NewInMemoryProfileRepository builds a repository holding the given profiles and no failure staged.
func NewInMemoryProfileRepository(profiles ...models.Profile) *InMemoryProfileRepository {
	stored := make(map[uuid.UUID]models.Profile, len(profiles))
	for _, profile := range profiles {
		stored[profile.ID] = profile
	}

	return &InMemoryProfileRepository{Profiles: stored}
}

// SaveProfile records the profile stamped with the wall clock as the database's NOW() would,
// or returns SaveFailure when one is staged.
func (r *InMemoryProfileRepository) SaveProfile(_ context.Context, newProfile models.NewProfile) (models.Profile, error) {
	if r.SaveFailure != nil {
		return models.Profile{}, r.SaveFailure
	}
	if _, exists := r.Profiles[newProfile.ID]; exists {
		return models.Profile{}, e.ErrProfileAlreadyExists{UserID: newProfile.ID}
	}

	savedAt := time.Now()
	saved := models.Profile{
		ID:              newProfile.ID,
		FirstName:       newProfile.FirstName,
		LastName:        newProfile.LastName,
		PictureUploadID: newProfile.PictureUploadID,
		CreatedAt:       savedAt,
		UpdatedAt:       savedAt,
	}
	r.Profiles[saved.ID] = saved

	return saved, nil
}

// UpdateProfile replaces the profile's names and picture, or returns UpdateFailure when one is staged.
func (r *InMemoryProfileRepository) UpdateProfile(
	_ context.Context,
	change models.UpdateProfileParams,
) (models.Profile, error) {
	if r.UpdateFailure != nil {
		return models.Profile{}, r.UpdateFailure
	}
	profile, exists := r.Profiles[change.ID]
	if !exists {
		return models.Profile{}, e.ErrProfileNotFound{UserID: change.ID}
	}

	profile.FirstName = change.FirstName
	profile.LastName = change.LastName
	profile.PictureUploadID = change.PictureUploadID
	profile.UpdatedAt = time.Now()
	r.Profiles[profile.ID] = profile

	return profile, nil
}

// GetProfileReadModelByID returns the client-facing view of the user's profile, or e.ErrProfileNotFound.
// It holds no uploads, so a picture carries its upload id but no URLs, unlike the uploads join in SQL.
func (r *InMemoryProfileRepository) GetProfileReadModelByID(
	_ context.Context,
	userID uuid.UUID,
) (readmodels.ProfileReadModel, error) {
	if r.FindFailure != nil {
		return readmodels.ProfileReadModel{}, r.FindFailure
	}
	profile, exists := r.Profiles[userID]
	if !exists {
		return readmodels.ProfileReadModel{}, e.ErrProfileNotFound{UserID: userID}
	}

	readModel := readmodels.ProfileReadModel{ID: profile.ID, FirstName: profile.FirstName, LastName: profile.LastName}
	if profile.PictureUploadID != uuid.Nil {
		pictureUploadID := profile.PictureUploadID
		readModel.Picture.UploadID = &pictureUploadID
	}

	return readModel, nil
}
