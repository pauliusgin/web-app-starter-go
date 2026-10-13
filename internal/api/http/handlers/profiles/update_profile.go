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

type updateProfileRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type updateProfileResponse struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
}

// UpdateProfileUseCase is the domain call the profile update handler depends on.
type UpdateProfileUseCase func(ctx context.Context, params usecases.UpdateProfileParams) (usecases.UpdateProfileResult, error)

// UpdateProfileDependencies carries the collaborator the profile update handler orchestrates.
type UpdateProfileDependencies struct {
	UpdateProfile UpdateProfileUseCase
}

// UpdateProfile returns a handler replacing both names on the authenticated user's profile.
func UpdateProfile(d UpdateProfileDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			slog.Error(handlers.RouteNotProtectedText, "path", r.URL.Path)
			handlers.WriteJSON(w, http.StatusInternalServerError,
				handlers.ErrorResponse{Code: handlers.ErrorCodeInternal, Message: unexpectedProfileUpdateFailureText})

			return
		}

		request, err := handlers.DecodeRequest[updateProfileRequest](w, r)
		if err != nil {
			writeUpdateProfileFailure(w, err)
			return
		}

		profileInput, err := validateProfileNames(input.ProfileInput{
			FirstName: request.FirstName,
			LastName:  request.LastName,
		})
		if err != nil {
			writeUpdateProfileFailure(w, err)
			return
		}

		result, err := d.UpdateProfile(r.Context(), usecases.UpdateProfileParams{
			UserID:    userID,
			FirstName: profileInput.FirstName,
			LastName:  profileInput.LastName,
		})
		if err != nil {
			writeUpdateProfileFailure(w, err)
			return
		}

		handlers.WriteJSON(w, http.StatusOK, updateProfileResponse{
			ID:        result.Profile.ID,
			FirstName: result.Profile.FirstName,
			LastName:  result.Profile.LastName,
		})
	}
}

var updateProfileRules = []handlers.RejectionRule{
	handlers.RejectWhen[profileerrors.ErrProfileNotFound](
		handlers.Rejection{Status: http.StatusNotFound, Code: handlers.ErrorCodeProfileNotFound}),
}

func writeUpdateProfileFailure(w http.ResponseWriter, err error) {
	handlers.WriteRejection(w, handlers.WriteRejectionParams{
		Err:                   err,
		RuleSets:              [][]handlers.RejectionRule{handlers.RequestDecodingRules, profileValidationRules, updateProfileRules},
		UnexpectedFailureText: unexpectedProfileUpdateFailureText,
	})
}
