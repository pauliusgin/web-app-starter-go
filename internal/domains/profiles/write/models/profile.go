// Package models defines the personal details a user keeps in their profile.
package models

import (
	"time"

	"github.com/google/uuid"
)

// Profile is a user's personal details as recorded in the profiles table; its ID is the user's ID.
type Profile struct {
	ID              uuid.UUID
	FirstName       string
	LastName        string
	PictureUploadID uuid.UUID
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewProfile is a profile about to be recorded for the user sharing its ID; a uuid.Nil PictureUploadID is no picture.
type NewProfile struct {
	ID              uuid.UUID
	FirstName       string
	LastName        string
	PictureUploadID uuid.UUID
}

// UpdateProfileParams are the names and picture replacing those of the profile with the given ID; a uuid.Nil
// PictureUploadID clears the picture.
type UpdateProfileParams struct {
	ID              uuid.UUID
	FirstName       string
	LastName        string
	PictureUploadID uuid.UUID
}
