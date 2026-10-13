package storage

import (
	"context"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/errors"
)

const inMemoryURLScheme = "memory://"

var _ FileStorage = (*InMemoryFileStorage)(nil)

// InMemoryFileStorage is a FileStorage that keeps stored files in a map keyed by path.
type InMemoryFileStorage struct {
	Files   map[string]StoreFileParams
	Deleted []string

	// StoreFailure, when set, is returned by every store after the first StoresBeforeFailure succeed.
	StoreFailure        error
	StoresBeforeFailure int
	// DeleteFailure, when set, is returned instead of deleting the file.
	DeleteFailure error

	storeCalls int
}

// NewInMemoryFileStorage builds a storage holding no files and no failure staged.
func NewInMemoryFileStorage() *InMemoryFileStorage {
	return &InMemoryFileStorage{Files: map[string]StoreFileParams{}}
}

// StoreFile keeps the file and answers with InMemoryFileURL of its path, unless a failure is due.
func (s *InMemoryFileStorage) StoreFile(_ context.Context, params StoreFileParams) (StoredFile, error) {
	s.storeCalls++
	if s.StoreFailure != nil && s.storeCalls > s.StoresBeforeFailure {
		return StoredFile{}, s.StoreFailure
	}

	s.Files[params.Path] = params

	return StoredFile{URL: InMemoryFileURL(params.Path)}, nil
}

// DeleteFile removes the file at path, or reports ErrStoredFileNotFound when none is kept there.
func (s *InMemoryFileStorage) DeleteFile(_ context.Context, path string) error {
	if s.DeleteFailure != nil {
		return s.DeleteFailure
	}
	if _, stored := s.Files[path]; !stored {
		return e.ErrStoredFileNotFound{Path: path}
	}

	delete(s.Files, path)
	s.Deleted = append(s.Deleted, path)

	return nil
}

// InMemoryFileURL is the URL the in-memory storage answers a stored path with.
func InMemoryFileURL(path string) string {
	return inMemoryURLScheme + path
}
