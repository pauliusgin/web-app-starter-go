// Package models holds the upload shapes the read side returns to clients.
package models

import "github.com/google/uuid"

// UploadReadModel is a stored file as clients see it; PreviewURL is nil when the file has no preview,
// and OriginalURL is nil when its purpose keeps only the preview.
type UploadReadModel struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Purpose     string
	Kind        string
	MimeType    string
	OriginalURL *string
	PreviewURL  *string
}

// GetUploadReadModelByIDParams names an upload and the user it must belong to.
type GetUploadReadModelByIDParams struct {
	ID     uuid.UUID
	UserID uuid.UUID
}
