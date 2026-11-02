package input

import (
	"strings"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
)

// GoogleIDTokenInput carries the ID token a google signup or sign-in sends.
type GoogleIDTokenInput struct {
	IDToken string
}

// ValidateGoogleIDTokenInput trims the ID token and checks one arrived; google checks the rest.
func ValidateGoogleIDTokenInput(rawInput GoogleIDTokenInput) (GoogleIDTokenInput, error) {
	googleIDTokenInput := GoogleIDTokenInput{IDToken: strings.TrimSpace(rawInput.IDToken)}

	if err := primitive.RequiredString(googleIDTokenInput.IDToken); err != nil {
		return GoogleIDTokenInput{}, primitive.ErrGoogleIDTokenMissing{}
	}

	return googleIDTokenInput, nil
}
