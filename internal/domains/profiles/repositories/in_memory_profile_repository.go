package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/models"
)

var _ ProfileRepository = (*InMemoryProfileRepository)(nil)

// InMemoryProfileRepository is a ProfileRepository that keeps profiles in a map keyed by user ID.
type InMemoryProfileRepository struct {
	Profiles map[uuid.UUID]models.Profile

	// SaveFailure and UpdateFailure, when set, are returned instead of doing the work.
	SaveFailure   error
	UpdateFailure error
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
		ID:        newProfile.ID,
		FirstName: newProfile.FirstName,
		LastName:  newProfile.LastName,
		CreatedAt: savedAt,
		UpdatedAt: savedAt,
	}
	r.Profiles[saved.ID] = saved

	return saved, nil
}

// UpdateProfile replaces the profile's names, or returns UpdateFailure when one is staged.
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
	profile.UpdatedAt = time.Now()
	r.Profiles[profile.ID] = profile

	return profile, nil
}
