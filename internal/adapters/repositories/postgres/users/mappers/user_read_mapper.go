package mappers

import (
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlnull"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/models"
)

// UserReadModelFromIDMapper maps a GetUserReadModelByID row; NULL columns become nil pointers
// and the credential flags become the user's sign-in methods.
func UserReadModelFromIDMapper(row sqlc.GetUserReadModelByIDRow) models.UserReadModel {
	mapped := models.UserReadModel{
		ID:              row.ID,
		Email:           row.Email,
		CreatedAt:       row.CreatedAt,
		EmailVerifiedAt: sqlnull.TimePointer(row.EmailVerifiedAt),
		SigninAt:        sqlnull.TimePointer(row.SigninAt),
		FirstName:       sqlnull.StringPointer(row.FirstName),
		LastName:        sqlnull.StringPointer(row.LastName),
		SignInMethods: models.NewSignInMethods(models.SignInMethodsParams{
			HasPassword:      row.HasPassword,
			HasGoogleSubject: row.HasGoogleSubject,
		}),
	}

	return mapped
}
