package repositories

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/errors"
	readmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/read/models"
	read "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/read/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/models"
)

var _ UploadWriteRepository = (*InMemoryUploadRepository)(nil)
var _ read.UploadReadRepository = (*InMemoryUploadRepository)(nil)

// InMemoryUploadRepository is both upload ports over one slice of uploads, kept in save order.
type InMemoryUploadRepository struct {
	Uploads []models.Upload

	// SaveFailure, ListFailure, DeleteFailure and FindFailure, when set, are returned instead of doing the work.
	SaveFailure   error
	ListFailure   error
	DeleteFailure error
	FindFailure   error
}

// NewInMemoryUploadRepository builds a repository holding the given uploads and no failure staged.
func NewInMemoryUploadRepository(uploads ...models.Upload) *InMemoryUploadRepository {
	return &InMemoryUploadRepository{Uploads: uploads}
}

// SaveUpload records the upload as pending, stamped with the wall clock as the database's NOW()
// would, or returns SaveFailure when one is staged.
func (r *InMemoryUploadRepository) SaveUpload(_ context.Context, newUpload models.NewUpload) (models.Upload, error) {
	if r.SaveFailure != nil {
		return models.Upload{}, r.SaveFailure
	}

	savedAt := time.Now()
	saved := models.Upload{
		ID:           newUpload.ID,
		UserID:       newUpload.UserID,
		Purpose:      newUpload.Purpose,
		Kind:         newUpload.Kind,
		MimeType:     newUpload.MimeType,
		SizeBytes:    newUpload.SizeBytes,
		Width:        newUpload.Width,
		Height:       newUpload.Height,
		OriginalPath: newUpload.OriginalPath,
		OriginalURL:  newUpload.OriginalURL,
		PreviewPath:  newUpload.PreviewPath,
		PreviewURL:   newUpload.PreviewURL,
		CreatedAt:    savedAt,
		UpdatedAt:    savedAt,
	}
	r.Uploads = append(r.Uploads, saved)

	return saved, nil
}

// ListUnusedPendingUploads returns unattached uploads created before cutoff, oldest first.
func (r *InMemoryUploadRepository) ListUnusedPendingUploads(
	_ context.Context,
) ([]models.Upload, error) {
	if r.ListFailure != nil {
		return nil, r.ListFailure
	}
	cutoff := time.Now().Add(-models.PendingUploadRetention)

	var pending []models.Upload
	for _, upload := range r.Uploads {
		if !upload.IsAttached() && upload.CreatedAt.Before(cutoff) {
			pending = append(pending, upload)
		}
	}

	slices.SortStableFunc(pending, func(a, b models.Upload) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})

	return pending, nil
}

// DeleteUploadByID removes the upload, or returns DeleteFailure when one is staged.
func (r *InMemoryUploadRepository) DeleteUploadByID(_ context.Context, id uuid.UUID) error {
	if r.DeleteFailure != nil {
		return r.DeleteFailure
	}

	r.Uploads = slices.DeleteFunc(r.Uploads, func(upload models.Upload) bool {
		return upload.ID == id
	})

	return nil
}

// GetUploadReadModelByID returns the client-facing view of the upload owned by the user, or e.ErrUploadNotFound.
func (r *InMemoryUploadRepository) GetUploadReadModelByID(
	_ context.Context,
	params readmodels.GetUploadReadModelByIDParams,
) (readmodels.UploadReadModel, error) {
	if r.FindFailure != nil {
		return readmodels.UploadReadModel{}, r.FindFailure
	}

	index := slices.IndexFunc(r.Uploads, func(upload models.Upload) bool {
		return upload.ID == params.ID && upload.UserID == params.UserID
	})
	if index < 0 {
		return readmodels.UploadReadModel{}, e.ErrUploadNotFound{ID: params.ID}
	}

	return uploadReadModel(r.Uploads[index]), nil
}

func uploadReadModel(upload models.Upload) readmodels.UploadReadModel {
	readModel := readmodels.UploadReadModel{
		ID:          upload.ID,
		UserID:      upload.UserID,
		Purpose:     string(upload.Purpose),
		Kind:        string(upload.Kind),
		MimeType:    upload.MimeType,
		OriginalURL: upload.OriginalURL,
	}
	if upload.PreviewURL != "" {
		previewURL := upload.PreviewURL
		readModel.PreviewURL = &previewURL
	}

	return readModel
}
