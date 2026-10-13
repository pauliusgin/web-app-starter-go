package mappers

import (
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlnull"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/models"
)

// UserReadModelFromIDMapper maps the FindUserReadModelByID row onto the read model; NULL → nil.
func UserReadModelFromIDMapper(row sqlc.FindUserReadModelByIDRow) models.UserReadModel {
	mapped := models.UserReadModel{
		ID:              row.ID,
		Email:           row.Email,
		CreatedAt:       row.CreatedAt,
		EmailVerifiedAt: sqlnull.TimePointer(row.EmailVerifiedAt),
		SigninAt:        sqlnull.TimePointer(row.SigninAt),
		FirstName:       sqlnull.StringPointer(row.FirstName),
		LastName:        sqlnull.StringPointer(row.LastName),
	}

	return mapped
}
