package input

import "github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"

// RequestEmailChangeInput carries the email change fields to validate.
type RequestEmailChangeInput struct {
	NewEmail string
}

// ValidateRequestEmailChangeInput normalizes the new email and checks it is a plain address.
func ValidateRequestEmailChangeInput(rawInput RequestEmailChangeInput) (RequestEmailChangeInput, error) {
	requestEmailChangeInput := RequestEmailChangeInput{NewEmail: primitive.NormalizeEmail(rawInput.NewEmail)}

	if err := primitive.ValidateEmail(requestEmailChangeInput.NewEmail); err != nil {
		return RequestEmailChangeInput{}, err
	}

	return requestEmailChangeInput, nil
}
