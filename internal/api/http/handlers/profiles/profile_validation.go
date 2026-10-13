// Package profiles holds the HTTP handlers for creating and updating the caller's profile.
package profiles

import (
	"net/http"
	"strings"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/input"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
)

// validateProfileNames trims and validates the names; they come back trimmed, as they are stored.
func validateProfileNames(rawNames input.ProfileInput) (input.ProfileInput, error) {
	profileInput := input.ProfileInput{
		FirstName: strings.TrimSpace(rawNames.FirstName),
		LastName:  strings.TrimSpace(rawNames.LastName),
	}
	if err := input.ValidateProfileInput(profileInput); err != nil {
		return input.ProfileInput{}, err
	}

	return profileInput, nil
}

var profileValidationRules = []handlers.RejectionRule{
	handlers.RejectWhen[primitive.ErrFirstNameMissing](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeFirstNameMissing}),
	handlers.RejectWhen[primitive.ErrLastNameMissing](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeLastNameMissing}),
}
