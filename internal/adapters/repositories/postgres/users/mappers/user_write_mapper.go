// Package mappers converts stored user rows into the user read and write models.
package mappers

import (
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
)

// CreateUserMapper maps the row CreateUser returns onto a user whose email is not yet verified.
func CreateUserMapper(row sqlc.CreateUserRow) models.User {
	mapped := models.User{
		ID:        row.ID,
		Email:     row.Email,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		SigninAt:  row.SigninAt.Time,
	}

	return mapped
}

// UserFromIDMapper maps the row FindUserByID returns onto the domain type.
func UserFromIDMapper(row sqlc.FindUserByIDRow) models.User {
	mapped := models.User{
		ID:              row.ID,
		Email:           row.Email,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		EmailVerifiedAt: row.EmailVerifiedAt.Time,
		SigninAt:        row.SigninAt.Time,
	}

	return mapped
}

// UserFromEmailMapper maps the row FindUserByEmail returns onto the domain type.
func UserFromEmailMapper(row sqlc.FindUserByEmailRow) models.User {
	mapped := models.User{
		ID:              row.ID,
		Email:           row.Email,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		EmailVerifiedAt: row.EmailVerifiedAt.Time,
		SigninAt:        row.SigninAt.Time,
	}

	return mapped
}

// MarkUserEmailAsVerifiedMapper maps the row MarkUserEmailAsVerified returns onto the domain type.
func MarkUserEmailAsVerifiedMapper(row sqlc.MarkUserEmailAsVerifiedRow) models.User {
	mapped := models.User{
		ID:              row.ID,
		Email:           row.Email,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		EmailVerifiedAt: row.EmailVerifiedAt.Time,
		SigninAt:        row.SigninAt.Time,
	}

	return mapped
}

// UserCredentialsFromEmailMapper maps the row FindUserCredentialsByEmail returns onto the domain type.
func UserCredentialsFromEmailMapper(row sqlc.FindUserCredentialsByEmailRow) models.UserCredentials {
	mapped := models.UserCredentials{
		ID:           row.ID,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
	}

	return mapped
}

// UserCredentialsFromIDMapper maps the row FindUserCredentialsByID returns onto the domain type.
func UserCredentialsFromIDMapper(row sqlc.FindUserCredentialsByIDRow) models.UserCredentials {
	mapped := models.UserCredentials{
		ID:           row.ID,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
	}

	return mapped
}

// MarkUserSignedInMapper maps the row MarkUserSignedIn returns onto the domain type.
func MarkUserSignedInMapper(row sqlc.MarkUserSignedInRow) models.User {
	mapped := models.User{
		ID:              row.ID,
		Email:           row.Email,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		EmailVerifiedAt: row.EmailVerifiedAt.Time,
		SigninAt:        row.SigninAt.Time,
	}

	return mapped
}

// ChangeUserEmailMapper maps the row ChangeUserEmail returns onto the domain type.
func ChangeUserEmailMapper(row sqlc.ChangeUserEmailRow) models.User {
	mapped := models.User{
		ID:              row.ID,
		Email:           row.Email,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		EmailVerifiedAt: row.EmailVerifiedAt.Time,
		SigninAt:        row.SigninAt.Time,
	}

	return mapped
}
