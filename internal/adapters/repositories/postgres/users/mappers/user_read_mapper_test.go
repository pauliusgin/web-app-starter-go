package mappers

import (
	"database/sql"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
)

func TestUserReadModelFromIDMapperMapsEmailVerifiedAt(t *testing.T) {
	mapped := UserReadModelFromIDMapper(sqlc.FindUserReadModelByIDRow{
		ID:              mappedUserID,
		Email:           mappedEmail,
		CreatedAt:       mappedCreatedAt,
		EmailVerifiedAt: sql.NullTime{Time: mappedVerifiedAt, Valid: true},
	})

	if mapped.EmailVerifiedAt == nil || !mapped.EmailVerifiedAt.Equal(mappedVerifiedAt) {
		t.Errorf("EmailVerifiedAt = %v, want %v", mapped.EmailVerifiedAt, mappedVerifiedAt)
	}
	if mapped.ID != mappedUserID {
		t.Errorf("ID = %s, want %s", mapped.ID, mappedUserID)
	}
}

func TestUserReadModelFromIDMapperMapsSigninAt(t *testing.T) {
	mapped := UserReadModelFromIDMapper(sqlc.FindUserReadModelByIDRow{
		ID:       mappedUserID,
		Email:    mappedEmail,
		SigninAt: sql.NullTime{Time: mappedSignedInAt, Valid: true},
	})

	if mapped.SigninAt == nil || !mapped.SigninAt.Equal(mappedSignedInAt) {
		t.Errorf("SigninAt = %v, want %v", mapped.SigninAt, mappedSignedInAt)
	}
}

func TestUserReadModelFromIDMapperMapsProfileNames(t *testing.T) {
	mapped := UserReadModelFromIDMapper(sqlc.FindUserReadModelByIDRow{
		ID:        mappedUserID,
		Email:     mappedEmail,
		FirstName: sql.NullString{String: "Ada", Valid: true},
		LastName:  sql.NullString{String: "Lovelace", Valid: true},
	})

	if mapped.FirstName == nil || *mapped.FirstName != "Ada" {
		t.Errorf("FirstName = %v, want %q", mapped.FirstName, "Ada")
	}
	if mapped.LastName == nil || *mapped.LastName != "Lovelace" {
		t.Errorf("LastName = %v, want %q", mapped.LastName, "Lovelace")
	}
}

func TestUserReadModelFromIDMapperLeavesNullColumnsNil(t *testing.T) {
	mapped := UserReadModelFromIDMapper(sqlc.FindUserReadModelByIDRow{
		ID:    mappedUserID,
		Email: mappedEmail,
	})

	if mapped.FirstName != nil || mapped.LastName != nil {
		t.Errorf("names = %v %v, want both nil without a profile", mapped.FirstName, mapped.LastName)
	}
	if mapped.EmailVerifiedAt != nil || mapped.SigninAt != nil {
		t.Errorf("timestamps = %v %v, want both nil", mapped.EmailVerifiedAt, mapped.SigninAt)
	}
}
