package profiles

import (
	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
)

// parsePictureUploadID reads an absent, null or blank id as no picture, which is uuid.Nil.
func parsePictureUploadID(rawPictureUploadID string) (uuid.UUID, error) {
	if rawPictureUploadID == "" {
		return uuid.Nil, nil
	}

	pictureUploadID, err := uuid.Parse(rawPictureUploadID)
	if err != nil {
		return uuid.Nil, primitive.ErrPictureUploadIDMalformed{RawPictureUploadID: rawPictureUploadID}
	}

	return pictureUploadID, nil
}
