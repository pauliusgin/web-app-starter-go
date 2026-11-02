// Package models defines the files users upload and the rules an upload must satisfy.
package models

import (
	"time"

	"github.com/google/uuid"
)

// Upload is a stored file as recorded in the uploads table.
type Upload struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Purpose      Purpose
	Kind         Kind
	MimeType     string
	SizeBytes    int64
	Width        int
	Height       int
	OriginalPath string
	OriginalURL  string
	PreviewPath  string
	PreviewURL   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	AttachedAt   time.Time
}

// IsAttached reports whether a resource has claimed the upload, sparing it from cleanup.
func (u Upload) IsAttached() bool {
	return !u.AttachedAt.IsZero()
}

// StoredPaths lists every file storage path the upload occupies, original first. A document has
// no preview and a preview-only image has no original, so either may be missing.
func (u Upload) StoredPaths() []string {
	paths := make([]string, 0, 2)
	for _, path := range []string{u.OriginalPath, u.PreviewPath} {
		if path != "" {
			paths = append(paths, path)
		}
	}

	return paths
}

// NewUpload is an upload whose files are stored, about to be recorded as pending.
type NewUpload struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Purpose      Purpose
	Kind         Kind
	MimeType     string
	SizeBytes    int64
	Width        int
	Height       int
	OriginalPath string
	OriginalURL  string
	PreviewPath  string
	PreviewURL   string
}

// AttachUploadParams names the upload a resource claims, and the owner and purpose it must have.
type AttachUploadParams struct {
	ID      uuid.UUID
	UserID  uuid.UUID
	Purpose Purpose
}
