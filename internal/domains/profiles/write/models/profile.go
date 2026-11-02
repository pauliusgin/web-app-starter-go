// Package models defines the personal details a user keeps in their profile.
package models

import (
	"time"

	"github.com/google/uuid"
)

// Profile is a user's personal details as recorded in the profiles table; its ID is the user's ID.
type Profile struct {
	ID        uuid.UUID
	FirstName string
	LastName  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewProfile is a profile about to be recorded for the user sharing its ID.
type NewProfile struct {
	ID        uuid.UUID
	FirstName string
	LastName  string
}

// UpdateProfileParams are those of the profile with the given ID.
type UpdateProfileParams struct {
	ID        uuid.UUID
	FirstName string
	LastName  string
}
