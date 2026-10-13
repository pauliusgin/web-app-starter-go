package mappers

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/models"
)

var uploadID = uuid.MustParse("6f1c2d3e-4b5a-4c6d-8e7f-9a0b1c2d3e4f")

var ownerID = uuid.MustParse("1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15")

var createdAt = time.Date(2026, time.October, 11, 9, 30, 0, 0, time.UTC)

var attachedAt = time.Date(2026, time.October, 11, 9, 45, 0, 0, time.UTC)

func TestUploadMapperKeepsEveryColumn(t *testing.T) {
	row := sqlc.Upload{
		ID:           uploadID,
		UserID:       ownerID,
		Purpose:      "profile_picture",
		Kind:         "image",
		MimeType:     "image/png",
		SizeBytes:    2048,
		Width:        sql.NullInt32{Int32: 800, Valid: true},
		Height:       sql.NullInt32{Int32: 600, Valid: true},
		OriginalPath: "a/original.png",
		OriginalUrl:  "https://files/a/original.png",
		PreviewPath:  sql.NullString{String: "a/preview.webp", Valid: true},
		PreviewUrl:   sql.NullString{String: "https://files/a/preview.webp", Valid: true},
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
		AttachedAt:   sql.NullTime{Time: attachedAt, Valid: true},
	}

	want := models.Upload{
		ID:           uploadID,
		UserID:       ownerID,
		Purpose:      models.PurposeProfilePicture,
		Kind:         models.KindImage,
		MimeType:     "image/png",
		SizeBytes:    2048,
		Width:        800,
		Height:       600,
		OriginalPath: "a/original.png",
		OriginalURL:  "https://files/a/original.png",
		PreviewPath:  "a/preview.webp",
		PreviewURL:   "https://files/a/preview.webp",
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
		AttachedAt:   attachedAt,
	}

	if mapped := UploadMapper(row); mapped != want {
		t.Errorf("UploadMapper = %+v\nwant          %+v", mapped, want)
	}
}

func TestUploadMapperTurnsNullColumnsIntoZeroValues(t *testing.T) {
	mapped := UploadMapper(sqlc.Upload{ID: uploadID, OriginalPath: "a/original.pdf"})

	if mapped.Width != 0 || mapped.Height != 0 {
		t.Errorf("dimensions = %dx%d, want 0x0", mapped.Width, mapped.Height)
	}
	if mapped.PreviewPath != "" || mapped.PreviewURL != "" {
		t.Errorf("preview = %q %q, want empty", mapped.PreviewPath, mapped.PreviewURL)
	}
	if mapped.IsAttached() {
		t.Error("an upload with NULL attached_at is attached, want pending")
	}
}

func TestSaveUploadParamsMapperStoresAbsentPreviewAndDimensionsAsNull(t *testing.T) {
	params := SaveUploadParamsMapper(models.NewUpload{
		ID:           uploadID,
		UserID:       ownerID,
		Purpose:      models.PurposeDocumentAttachment,
		Kind:         models.KindDocument,
		MimeType:     "application/pdf",
		SizeBytes:    4096,
		OriginalPath: "a/original.pdf",
		OriginalURL:  "https://files/a/original.pdf",
	})

	if params.Width.Valid || params.Height.Valid {
		t.Errorf("dimensions = %+v %+v, want NULL", params.Width, params.Height)
	}
	if params.PreviewPath.Valid || params.PreviewUrl.Valid {
		t.Errorf("preview = %+v %+v, want NULL", params.PreviewPath, params.PreviewUrl)
	}
	if params.Purpose != "document_attachment" || params.Kind != "document" {
		t.Errorf("params = %+v, want the purpose and kind carried over", params)
	}
}

func TestSaveUploadParamsMapperStoresImageDimensionsAndPreview(t *testing.T) {
	params := SaveUploadParamsMapper(models.NewUpload{
		Width:       800,
		Height:      600,
		PreviewPath: "a/preview.webp",
		PreviewURL:  "https://files/a/preview.webp",
	})

	if params.Width != (sql.NullInt32{Int32: 800, Valid: true}) || params.Height != (sql.NullInt32{Int32: 600, Valid: true}) {
		t.Errorf("dimensions = %+v %+v, want 800x600", params.Width, params.Height)
	}
	if params.PreviewPath != (sql.NullString{String: "a/preview.webp", Valid: true}) {
		t.Errorf("preview path = %+v, want a/preview.webp", params.PreviewPath)
	}
}
