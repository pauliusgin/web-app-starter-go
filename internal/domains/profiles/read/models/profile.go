// Package models holds the profile shapes the read side returns to clients.
package models

import "github.com/google/uuid"

// ProfileReadModel is a user's profile as clients see it; its ID is the user's ID.
type ProfileReadModel struct {
	ID        uuid.UUID
	FirstName string
	LastName  string
	Picture   Picture
}

// Picture is the upload a profile shows as its picture, with the URLs its files are served from;
// every field is nil when the profile has no picture.
type Picture struct {
	UploadID    *uuid.UUID
	PreviewURL  *string
	OriginalURL *string
}
