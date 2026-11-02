// Package input validates whole request payloads by combining primitive validators.
package input

import "github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"

// SignUpInput carries the signup fields to validate.
type SignUpInput struct {
	Email    string
	Password string
}

// ValidateSignUpInput normalizes the email, checks every signup field and reports the first one that fails.
func ValidateSignUpInput(rawInput SignUpInput) (SignUpInput, error) {
	signUpInput := SignUpInput{
		Email:    primitive.NormalizeEmail(rawInput.Email),
		Password: rawInput.Password,
	}

	if err := primitive.ValidateEmail(signUpInput.Email); err != nil {
		return SignUpInput{}, err
	}

	if err := primitive.ValidatePassword(signUpInput.Password); err != nil {
		return SignUpInput{}, err
	}

	return signUpInput, nil
}
