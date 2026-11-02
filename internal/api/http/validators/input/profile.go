package input

import (
	"strings"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
)

// ProfileInput carries the profile names to validate on creation or replacement.
type ProfileInput struct {
	FirstName string
	LastName  string
}

// ValidateProfileInput trims and checks both names; they come back as they are stored.
func ValidateProfileInput(rawInput ProfileInput) (ProfileInput, error) {
	profileInput := ProfileInput{
		FirstName: strings.TrimSpace(rawInput.FirstName),
		LastName:  strings.TrimSpace(rawInput.LastName),
	}

	if err := primitive.RequiredString(profileInput.FirstName); err != nil {
		return ProfileInput{}, primitive.ErrFirstNameMissing{}
	}
	if err := primitive.RequiredString(profileInput.LastName); err != nil {
		return ProfileInput{}, primitive.ErrLastNameMissing{}
	}

	return profileInput, nil
}
