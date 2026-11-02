package input

import "github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"

// PasswordResetEmailInput carries the email a forgotten password reset link is asked for.
type PasswordResetEmailInput struct {
	Email string
}

// ValidatePasswordResetEmailInput normalizes the email and checks it is a plain address.
func ValidatePasswordResetEmailInput(rawInput PasswordResetEmailInput) (PasswordResetEmailInput, error) {
	passwordResetEmailInput := PasswordResetEmailInput{Email: primitive.NormalizeEmail(rawInput.Email)}

	if err := primitive.ValidateEmail(passwordResetEmailInput.Email); err != nil {
		return PasswordResetEmailInput{}, err
	}

	return passwordResetEmailInput, nil
}
