package input

import "github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"

// SignInInput carries the sign-in fields to validate.
type SignInInput struct {
	Email    string
	Password string
}

// ValidateSignInInput normalizes the email, checks its shape and that a password arrived, reporting the first failure.
func ValidateSignInInput(rawInput SignInInput) (SignInInput, error) {
	signInInput := SignInInput{
		Email:    primitive.NormalizeEmail(rawInput.Email),
		Password: rawInput.Password,
	}

	if err := primitive.ValidateEmail(signInInput.Email); err != nil {
		return SignInInput{}, err
	}

	if err := primitive.RequiredString(signInInput.Password); err != nil {
		return SignInInput{}, primitive.ErrPasswordMissing{}
	}

	return signInInput, nil
}
