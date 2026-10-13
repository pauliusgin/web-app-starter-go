package input

import "github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"

// SignInInput carries the sign-in fields to validate.
type SignInInput struct {
	Email    string
	Password string
}

// ValidateSignInInput checks the email shape and that a password arrived, reporting the first failure.
func ValidateSignInInput(signInInput SignInInput) error {
	if err := primitive.ValidateEmail(signInInput.Email); err != nil {
		return err
	}

	if err := primitive.RequiredString(signInInput.Password); err != nil {
		return primitive.ErrPasswordMissing{}
	}

	return nil
}
