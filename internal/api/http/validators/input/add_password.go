package input

import "github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"

// AddPasswordInput carries the password a google-only account adds.
type AddPasswordInput struct {
	Password string
}

// ValidateAddPasswordInput checks the password against the password rules, verbatim and untrimmed.
func ValidateAddPasswordInput(addPasswordInput AddPasswordInput) (AddPasswordInput, error) {
	if err := primitive.ValidatePassword(addPasswordInput.Password); err != nil {
		return AddPasswordInput{}, err
	}

	return addPasswordInput, nil
}
