// Package mappers converts between profiles rows and the profile write and read models.
package mappers

import (
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlnull"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/models"
)

// ProfileMapper maps a stored profiles row onto the domain type.
func ProfileMapper(row sqlc.Profile) models.Profile {
	mapped := models.Profile{
		ID:              row.ID,
		FirstName:       row.FirstName,
		LastName:        row.LastName,
		PictureUploadID: row.PictureUploadID.UUID,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}

	return mapped
}

// SaveProfileParamsMapper maps a profile about to be recorded onto the insert query's parameters.
func SaveProfileParamsMapper(newProfile models.NewProfile) sqlc.SaveProfileParams {
	return sqlc.SaveProfileParams{
		ID:              newProfile.ID,
		FirstName:       newProfile.FirstName,
		LastName:        newProfile.LastName,
		PictureUploadID: sqlnull.UUIDUnlessNil(newProfile.PictureUploadID),
	}
}

// UpdateProfileParamsMapper maps a names and picture change onto the update query's parameters.
func UpdateProfileParamsMapper(change models.UpdateProfileParams) sqlc.UpdateProfileParams {
	return sqlc.UpdateProfileParams{
		ID:              change.ID,
		FirstName:       change.FirstName,
		LastName:        change.LastName,
		PictureUploadID: sqlnull.UUIDUnlessNil(change.PictureUploadID),
	}
}
