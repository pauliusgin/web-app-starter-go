package mappers

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/models"
)

var profileID = uuid.MustParse("3a7e1c9b-2d4f-4e6a-9b8c-1f2e3d4c5b6a")

var createdAt = time.Date(2026, time.October, 12, 9, 0, 0, 0, time.UTC)

var updatedAt = time.Date(2026, time.October, 12, 10, 15, 0, 0, time.UTC)

func TestProfileMapperKeepsEveryDomainColumn(t *testing.T) {
	row := sqlc.Profile{
		ID:        profileID,
		FirstName: "Ada",
		LastName:  "Lovelace",
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		DeletedAt: sql.NullTime{},
	}

	want := models.Profile{
		ID:        profileID,
		FirstName: "Ada",
		LastName:  "Lovelace",
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	if mapped := ProfileMapper(row); mapped != want {
		t.Errorf("ProfileMapper = %+v\nwant           %+v", mapped, want)
	}
}

func TestSaveProfileParamsMapperCarriesIDAndNames(t *testing.T) {
	newProfile := models.NewProfile{ID: profileID, FirstName: "Ada", LastName: "Lovelace"}

	want := sqlc.SaveProfileParams{ID: profileID, FirstName: "Ada", LastName: "Lovelace"}

	if mapped := SaveProfileParamsMapper(newProfile); mapped != want {
		t.Errorf("SaveProfileParamsMapper = %+v, want %+v", mapped, want)
	}
}

func TestUpdateProfileParamsMapperCarriesIDAndNames(t *testing.T) {
	change := models.UpdateProfileParams{ID: profileID, FirstName: "Augusta", LastName: "King"}

	want := sqlc.UpdateProfileParams{ID: profileID, FirstName: "Augusta", LastName: "King"}

	if mapped := UpdateProfileParamsMapper(change); mapped != want {
		t.Errorf("UpdateProfileParamsMapper = %+v, want %+v", mapped, want)
	}
}
