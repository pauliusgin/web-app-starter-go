package usecases

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/models"
	uploadmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/models"
	uploadrepositories "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/repositories"
)

var testUserID = uuid.MustParse("1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15")

var storedAt = time.Date(2026, time.October, 12, 9, 0, 0, 0, time.UTC)

var errDatabaseDown = errors.New("database unreachable")

func storedProfile() models.Profile {
	return models.Profile{
		ID:        testUserID,
		FirstName: "Ada",
		LastName:  "Lovelace",
		CreatedAt: storedAt,
		UpdatedAt: storedAt,
	}
}

var pendingPictureID = uuid.MustParse("7d2e5f3c-6a91-4b22-9c4d-3f8b0e7a5d26")

var attachmentID = uuid.MustParse("3a8c1d7e-2f64-4b9a-8e15-6d0c4b2a7f93")

// pendingPictureUploads holds one unattached profile picture of the test user and one of another purpose.
func pendingPictureUploads() *uploadrepositories.InMemoryUploadRepository {
	return uploadrepositories.NewInMemoryUploadRepository(
		uploadmodels.Upload{
			ID:          pendingPictureID,
			UserID:      testUserID,
			Purpose:     uploadmodels.PurposeProfilePicture,
			Kind:        uploadmodels.KindImage,
			PreviewPath: "picture/preview.webp",
			CreatedAt:   storedAt,
			UpdatedAt:   storedAt,
		},
		uploadmodels.Upload{
			ID:           attachmentID,
			UserID:       testUserID,
			Purpose:      uploadmodels.PurposeDocumentAttachment,
			Kind:         uploadmodels.KindDocument,
			OriginalPath: "attachment/original.pdf",
			CreatedAt:    storedAt,
			UpdatedAt:    storedAt,
		},
	)
}

func isAttached(uploads *uploadrepositories.InMemoryUploadRepository, uploadID uuid.UUID) bool {
	for _, upload := range uploads.Uploads {
		if upload.ID == uploadID {
			return upload.IsAttached()
		}
	}

	return false
}
