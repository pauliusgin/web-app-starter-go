package input

import "github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"

// ChangePasswordInput carries the password change fields to validate.
type ChangePasswordInput struct {
	CurrentPassword string
	NewPassword     string
}

// ValidateChangePasswordInput checks both passwords against their rules, verbatim and untrimmed.
func ValidateChangePasswordInput(changePasswordInput ChangePasswordInput) (ChangePasswordInput, error) {
	if err := primitive.RequiredString(changePasswordInput.CurrentPassword); err != nil {
		return ChangePasswordInput{}, primitive.ErrPasswordMissing{}
	}

	if err := primitive.ValidatePassword(changePasswordInput.NewPassword); err != nil {
		return ChangePasswordInput{}, err
	}

	return changePasswordInput, nil
}
