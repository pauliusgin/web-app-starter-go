package input

import "github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"

// ChangePasswordInput carries the password change fields to validate.
type ChangePasswordInput struct {
	CurrentPassword string
	NewPassword     string
}

// ValidateChangePasswordInput checks a current password arrived and the new one meets the password rules.
func ValidateChangePasswordInput(changePasswordInput ChangePasswordInput) error {
	if err := primitive.RequiredString(changePasswordInput.CurrentPassword); err != nil {
		return primitive.ErrPasswordMissing{}
	}

	return primitive.ValidatePassword(changePasswordInput.NewPassword)
}
