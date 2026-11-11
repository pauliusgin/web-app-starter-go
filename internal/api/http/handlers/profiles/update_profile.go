package profiles

import (
	"context"
	"log/slog"
	"net/http"

	readmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/read/models"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/input"
	profileerrors "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/read/queries"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/usecases"
)

type updateProfileRequest struct {
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	PictureUploadID string `json:"pictureUploadId"`
}

type updateProfileResponse struct {
	readmodels.ProfileReadModel
}

// UpdateProfileUseCase is the domain call the profile update handler depends on.
type UpdateProfileUseCase func(ctx context.Context, params usecases.UpdateProfileParams) (usecases.UpdateProfileResult, error)

// UpdateProfileDependencies carries the use case replacing the profile and the query re-reading it.
type UpdateProfileDependencies struct {
	UpdateProfile  UpdateProfileUseCase
	GetProfileByID GetProfileByIDUseCase
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

		profileInput, err := input.ValidateProfileInput(input.ProfileInput{
			FirstName:       request.FirstName,
			LastName:        request.LastName,
			PictureUploadID: request.PictureUploadID,
		})
		if err != nil {
			writeUpdateProfileFailure(w, err)
			return
		}

		pictureUploadID, err := parsePictureUploadID(profileInput.PictureUploadID)
		if err != nil {
			writeUpdateProfileFailure(w, err)
			return
		}

		result, err := d.UpdateProfile(r.Context(), usecases.UpdateProfileParams{
			UserID:          userID,
			FirstName:       profileInput.FirstName,
			LastName:        profileInput.LastName,
			PictureUploadID: pictureUploadID,
		})
		if err != nil {
			writeUpdateProfileFailure(w, err)
			return
		}

		readResult, err := d.GetProfileByID(r.Context(), queries.GetProfileByIDParams{UserID: result.Profile.ID})
		if err != nil {
			writeUpdateProfileFailure(w, err)
			return
		}

		handlers.WriteJSON(w, http.StatusOK, buildUpdateProfileResponse(readResult.Profile))
	}
}

var updateProfileRules = []handlers.RejectionRule{
	handlers.RejectWhen[profileerrors.ErrProfileNotFound](
		handlers.Rejection{Status: http.StatusNotFound, Code: handlers.ErrorCodeProfileNotFound}),
}

func buildUpdateProfileResponse(profile readmodels.ProfileReadModel) updateProfileResponse {
	return updateProfileResponse{ProfileReadModel: profile}
}

func writeUpdateProfileFailure(w http.ResponseWriter, err error) {
	handlers.WriteRejection(w, handlers.WriteRejectionParams{
		Err: err,
		RuleSets: [][]handlers.RejectionRule{
			handlers.RequestDecodingRules, profileValidationRules, profilePictureRules, updateProfileRules,
		},
		UnexpectedFailureText: unexpectedProfileUpdateFailureText,
	})
}
