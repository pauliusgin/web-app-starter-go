package input

import "github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"

// RequestEmailChangeInput carries the email change fields to validate.
type RequestEmailChangeInput struct {
	NewEmail string
}

// ValidateRequestEmailChangeInput checks the new email is a plain address.
func ValidateRequestEmailChangeInput(requestEmailChangeInput RequestEmailChangeInput) error {
	return primitive.ValidateEmail(requestEmailChangeInput.NewEmail)
}
