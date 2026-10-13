// Package mappers converts between profiles rows and domain profiles.
package mappers

import (
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/models"
)

// ProfileMapper maps a stored profiles row onto the domain type.
func ProfileMapper(row sqlc.Profile) models.Profile {
	mapped := models.Profile{
		ID:        row.ID,
		FirstName: row.FirstName,
		LastName:  row.LastName,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}

	return mapped
}

// SaveProfileParamsMapper maps a profile about to be recorded onto the insert query's parameters.
func SaveProfileParamsMapper(newProfile models.NewProfile) sqlc.SaveProfileParams {
	return sqlc.SaveProfileParams{
		ID:        newProfile.ID,
		FirstName: newProfile.FirstName,
		LastName:  newProfile.LastName,
	}
}

// UpdateProfileParamsMapper maps a name change onto the update query's parameters.
func UpdateProfileParamsMapper(change models.UpdateProfileParams) sqlc.UpdateProfileParams {
	return sqlc.UpdateProfileParams{
		ID:        change.ID,
		FirstName: change.FirstName,
		LastName:  change.LastName,
	}
}
