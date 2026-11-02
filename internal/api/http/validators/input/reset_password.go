package input

import (
	"strings"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
)

// ResetPasswordInput carries the reset link's token and the new password to validate.
type ResetPasswordInput struct {
	Token       string
	NewPassword string
}

// ValidateResetPasswordInput trims the token and checks one arrived, then checks the new password verbatim.
func ValidateResetPasswordInput(rawInput ResetPasswordInput) (ResetPasswordInput, error) {
	resetPasswordInput := ResetPasswordInput{
		Token:       strings.TrimSpace(rawInput.Token),
		NewPassword: rawInput.NewPassword,
	}

	if err := primitive.RequiredString(resetPasswordInput.Token); err != nil {
		return ResetPasswordInput{}, primitive.ErrPasswordResetTokenMissing{}
	}

	if err := primitive.ValidatePassword(resetPasswordInput.NewPassword); err != nil {
		return ResetPasswordInput{}, err
	}

	return resetPasswordInput, nil
}
