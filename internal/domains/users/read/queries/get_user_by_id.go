// Package queries holds the read-side lookups answering clients with user read models.
package queries

import (
	"context"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/repositories"
)

// GetUserByIDParams is the user being looked up.
type GetUserByIDParams struct {
	UserID uuid.UUID
}

// GetUserByIDResult is the client-facing view of the looked-up user.
type GetUserByIDResult struct {
	User models.UserReadModel
}

// GetUserByIDDependencies carries the user read port the lookup goes through.
type GetUserByIDDependencies struct {
	UserReadRepository repositories.UserRepository
}

// GetUserByID loads the client-facing view of one live user.
func GetUserByID(ctx context.Context, d GetUserByIDDependencies, params GetUserByIDParams) (GetUserByIDResult, error) {
	user, err := d.UserReadRepository.GetUserReadModelByID(ctx, params.UserID)
	if err != nil {
		return GetUserByIDResult{}, err
	}

	return GetUserByIDResult{User: user}, nil
}
