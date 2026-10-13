// Package mappers converts between uploads rows and domain uploads.
package mappers

import (
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlnull"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/models"
)

// UploadMapper maps a stored uploads row onto the domain type; NULL columns become zero values.
func UploadMapper(row sqlc.Upload) models.Upload {
	mapped := models.Upload{
		ID:           row.ID,
		UserID:       row.UserID,
		Purpose:      models.Purpose(row.Purpose),
		Kind:         models.Kind(row.Kind),
		MimeType:     row.MimeType,
		SizeBytes:    row.SizeBytes,
		Width:        int(row.Width.Int32),
		Height:       int(row.Height.Int32),
		OriginalPath: row.OriginalPath,
		OriginalURL:  row.OriginalUrl,
		PreviewPath:  row.PreviewPath.String,
		PreviewURL:   row.PreviewUrl.String,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
		AttachedAt:   row.AttachedAt.Time,
	}

	return mapped
}

// SaveUploadParamsMapper maps an upload about to be recorded onto the insert query's parameters,
// storing absent dimensions and preview as NULL.
func SaveUploadParamsMapper(newUpload models.NewUpload) sqlc.SaveUploadParams {
	return sqlc.SaveUploadParams{
		ID:           newUpload.ID,
		UserID:       newUpload.UserID,
		Purpose:      string(newUpload.Purpose),
		Kind:         string(newUpload.Kind),
		MimeType:     newUpload.MimeType,
		SizeBytes:    newUpload.SizeBytes,
		Width:        sqlnull.Int32UnlessZero(newUpload.Width),
		Height:       sqlnull.Int32UnlessZero(newUpload.Height),
		OriginalPath: newUpload.OriginalPath,
		OriginalUrl:  newUpload.OriginalURL,
		PreviewPath:  sqlnull.StringUnlessEmpty(newUpload.PreviewPath),
		PreviewUrl:   sqlnull.StringUnlessEmpty(newUpload.PreviewURL),
	}
}
