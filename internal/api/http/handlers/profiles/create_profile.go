package profiles

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/input"
	profileerrors "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/usecases"
)

type createProfileRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type createProfileResponse struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
}

// CreateProfileUseCase is the domain call the profile creation handler depends on.
type CreateProfileUseCase func(ctx context.Context, params usecases.CreateProfileParams) (usecases.CreateProfileResult, error)

// CreateProfileDependencies carries the collaborator the profile creation handler orchestrates.
type CreateProfileDependencies struct {
	CreateProfile CreateProfileUseCase
}

// CreateProfile returns a handler recording the authenticated user's profile.
func CreateProfile(d CreateProfileDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			slog.Error(handlers.RouteNotProtectedText, "path", r.URL.Path)
			handlers.WriteJSON(w, http.StatusInternalServerError,
				handlers.ErrorResponse{Code: handlers.ErrorCodeInternal, Message: unexpectedProfileCreationFailureText})

			return
		}

		request, err := handlers.DecodeRequest[createProfileRequest](w, r)
		if err != nil {
			writeCreateProfileFailure(w, err)
			return
		}

		profileInput, err := validateProfileNames(input.ProfileInput{
			FirstName: request.FirstName,
			LastName:  request.LastName,
		})
		if err != nil {
			writeCreateProfileFailure(w, err)
			return
		}

		result, err := d.CreateProfile(r.Context(), usecases.CreateProfileParams{
			UserID:    userID,
			FirstName: profileInput.FirstName,
			LastName:  profileInput.LastName,
		})
		if err != nil {
			writeCreateProfileFailure(w, err)
			return
		}

		handlers.WriteJSON(w, http.StatusCreated, createProfileResponse{
			ID:        result.Profile.ID,
			FirstName: result.Profile.FirstName,
			LastName:  result.Profile.LastName,
		})
	}
}

var createProfileRules = []handlers.RejectionRule{
	handlers.RejectWhen[profileerrors.ErrProfileAlreadyExists](
		handlers.Rejection{Status: http.StatusConflict, Code: handlers.ErrorCodeProfileExists}),
}

func writeCreateProfileFailure(w http.ResponseWriter, err error) {
	handlers.WriteRejection(w, handlers.WriteRejectionParams{
		Err:                   err,
		RuleSets:              [][]handlers.RejectionRule{handlers.RequestDecodingRules, profileValidationRules, createProfileRules},
		UnexpectedFailureText: unexpectedProfileCreationFailureText,
	})
}
