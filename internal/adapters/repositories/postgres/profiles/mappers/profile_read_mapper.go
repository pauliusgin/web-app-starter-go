package mappers

import (
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/read/models"
)

// ProfileReadModelFromIDMapper maps the single profile lookup row onto the read model.
func ProfileReadModelFromIDMapper(row sqlc.GetProfileReadModelByIDRow) models.ProfileReadModel {
	return models.ProfileReadModel{
		ID:        row.ID,
		FirstName: row.FirstName,
		LastName:  row.LastName,
	}
}
