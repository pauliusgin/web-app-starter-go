// Package queries holds the read-side lookups answering clients with profile read models.
package queries

import (
	"context"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/read/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/read/repositories"
)

// GetProfileByIDParams is the authenticated user whose profile to return.
type GetProfileByIDParams struct {
	UserID uuid.UUID
}

// GetProfileByIDResult is the client-facing view of the user's profile.
type GetProfileByIDResult struct {
	Profile models.ProfileReadModel
}

// GetProfileByIDDependencies carries the profile read port the lookup goes through.
type GetProfileByIDDependencies struct {
	ProfileReadRepository repositories.ProfileReadRepository
}

// GetProfileByID returns the user's own profile, or errors.ErrProfileNotFound.
func GetProfileByID(
	ctx context.Context,
	d GetProfileByIDDependencies,
	params GetProfileByIDParams,
) (GetProfileByIDResult, error) {
	profile, err := d.ProfileReadRepository.GetProfileReadModelByID(ctx, params.UserID)
	if err != nil {
		return GetProfileByIDResult{}, err
	}

	return GetProfileByIDResult{Profile: profile}, nil
}
