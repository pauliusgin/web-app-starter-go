// Package models holds the profile shapes the read side returns to clients.
package models

import "github.com/google/uuid"

// ProfileReadModel is a user's profile as clients see it; its ID is the user's ID.
type ProfileReadModel struct {
	ID        uuid.UUID
	FirstName string
	LastName  string
}
