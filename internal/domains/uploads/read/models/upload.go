// Package models holds the upload shapes the read side returns to clients.
package models

import "github.com/google/uuid"

// UploadReadModel is a stored file as clients see it; PreviewURL is nil when the file has no preview,
// and OriginalURL is nil when its purpose keeps only the preview.
// It is also the HTTP response body: json keys are API contract, and a missing detail encodes as null.
type UploadReadModel struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"userId"`
	Purpose     string    `json:"purpose"`
	Kind        string    `json:"kind"`
	MimeType    string    `json:"mimeType"`
	OriginalURL *string   `json:"originalUrl"`
	PreviewURL  *string   `json:"previewUrl"`
}

// GetUploadReadModelByIDParams names an upload and the user it must belong to.
type GetUploadReadModelByIDParams struct {
	ID     uuid.UUID
	UserID uuid.UUID
}
