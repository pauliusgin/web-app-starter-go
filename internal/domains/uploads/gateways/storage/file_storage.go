// Package storage declares the outbound port uploaded files are stored in and deleted from.
package storage

import "context"

// StoreFileParams is the content to store, where to store it and the type it is served with.
type StoreFileParams struct {
	Path        string
	ContentType string
	Content     []byte
}

// StoredFile is where a stored file can be downloaded from.
type StoredFile struct {
	URL string
}

// FileStorage is the outbound file port the domain calls and a storage provider adapter implements.
type FileStorage interface {
	StoreFile(ctx context.Context, params StoreFileParams) (StoredFile, error)

	// DeleteFile removes the file at path, reporting ErrStoredFileNotFound when nothing is there.
	DeleteFile(ctx context.Context, path string) error
}
