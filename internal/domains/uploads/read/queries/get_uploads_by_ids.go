// Package queries holds the read-side lookups answering clients with upload read models.
package queries

import (
	"context"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/read/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/read/repositories"
)

// GetUploadsByIDsParams is the authenticated user and the uploads of theirs to return.
type GetUploadsByIDsParams struct {
	UserID    uuid.UUID
	UploadIDs []uuid.UUID
}

// GetUploadsByIDsResult is the client-facing view of the user's uploads, in the order they were asked for.
type GetUploadsByIDsResult struct {
	Uploads []models.UploadReadModel
}

// GetUploadsByIDsDependencies carries the upload read port the lookup goes through.
type GetUploadsByIDsDependencies struct {
	UploadReadRepository repositories.UploadReadRepository
}

// GetUploadsByIDs returns the user's own uploads in the order asked for, or errors.ErrUploadNotFound for the
// first that is missing or someone else's. One lookup per upload: an upload request carries at most a few files.
func GetUploadsByIDs(
	ctx context.Context,
	d GetUploadsByIDsDependencies,
	params GetUploadsByIDsParams,
) (GetUploadsByIDsResult, error) {
	uploads := make([]models.UploadReadModel, 0, len(params.UploadIDs))
	for _, uploadID := range params.UploadIDs {
		upload, err := d.UploadReadRepository.GetUploadReadModelByID(ctx, models.GetUploadReadModelByIDParams{
			ID:     uploadID,
			UserID: params.UserID,
		})
		if err != nil {
			return GetUploadsByIDsResult{}, err
		}
		uploads = append(uploads, upload)
	}

	return GetUploadsByIDsResult{Uploads: uploads}, nil
}
