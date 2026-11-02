package mappers

import (
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlnull"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/read/models"
)

// UploadReadModelFromIDMapper maps the single upload lookup row onto the read model; a NULL preview becomes nil.
func UploadReadModelFromIDMapper(row sqlc.GetUploadReadModelByIDRow) models.UploadReadModel {
	return models.UploadReadModel{
		ID:          row.ID,
		UserID:      row.UserID,
		Purpose:     row.Purpose,
		Kind:        row.Kind,
		MimeType:    row.MimeType,
		OriginalURL: row.OriginalUrl,
		PreviewURL:  sqlnull.StringPointer(row.PreviewUrl),
	}
}
