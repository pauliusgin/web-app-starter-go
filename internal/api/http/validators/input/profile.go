package input

import "github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"

// ProfileInput carries the profile names to validate, shared by profile creation and update.
type ProfileInput struct {
	FirstName string
	LastName  string
}

// ValidateProfileInput checks both names arrived and are not blank.
func ValidateProfileInput(profileInput ProfileInput) error {
	if err := primitive.RequiredString(profileInput.FirstName); err != nil {
		return primitive.ErrFirstNameMissing{}
	}
	if err := primitive.RequiredString(profileInput.LastName); err != nil {
		return primitive.ErrLastNameMissing{}
	}

	return nil
}
