// Package models holds the profile shapes the read side returns to clients.
package models

import "github.com/google/uuid"

// ProfileReadModel is a user's profile as clients see it; its ID is the user's ID.
// It is also the HTTP response body: json keys are API contract, and a missing detail encodes as null.
type ProfileReadModel struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Picture   Picture   `json:"picture"`
}

// Picture is the upload a profile shows as its picture, with the URLs its files are served from;
// every field is nil when the profile has no picture.
type Picture struct {
	UploadID    *uuid.UUID `json:"uploadId"`
	PreviewURL  *string    `json:"previewUrl"`
	OriginalURL *string    `json:"originalUrl"`
}
