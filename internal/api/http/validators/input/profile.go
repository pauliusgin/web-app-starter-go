package input

import (
	"strings"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
)

// ProfileInput carries the profile names and picture upload id to validate on creation or replacement.
type ProfileInput struct {
	FirstName       string
	LastName        string
	PictureUploadID string
}

// ValidateProfileInput trims, NFC-normalizes and checks both names, and trims the picture upload id; they come
// back as they are stored.
func ValidateProfileInput(rawInput ProfileInput) (ProfileInput, error) {
	profileInput := ProfileInput{
		FirstName:       normalizePersonName(rawInput.FirstName),
		LastName:        normalizePersonName(rawInput.LastName),
		PictureUploadID: strings.TrimSpace(rawInput.PictureUploadID),
	}

	if err := primitive.RequiredString(profileInput.FirstName); err != nil {
		return ProfileInput{}, primitive.ErrFirstNameMissing{}
	}
	if err := primitive.RequiredString(profileInput.LastName); err != nil {
		return ProfileInput{}, primitive.ErrLastNameMissing{}
	}
	if err := primitive.ValidatePersonName(profileInput.FirstName); err != nil {
		return ProfileInput{}, primitive.ErrFirstNameInvalid{Cause: err}
	}
	if err := primitive.ValidatePersonName(profileInput.LastName); err != nil {
		return ProfileInput{}, primitive.ErrLastNameInvalid{Cause: err}
	}

	return profileInput, nil
}
