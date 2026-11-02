// Package profiles holds the HTTP handlers for creating and updating the caller's profile.
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
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/read/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/read/queries"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/usecases"
)

type createProfileRequest struct {
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	PictureUploadID string `json:"pictureUploadId"`
}

type createProfileResponse struct {
	ID        uuid.UUID                    `json:"id"`
	FirstName string                       `json:"firstName"`
	LastName  string                       `json:"lastName"`
	Picture   createProfilePictureResponse `json:"picture"`
}

type createProfilePictureResponse struct {
	UploadID    *uuid.UUID `json:"uploadId"`
	PreviewURL  *string    `json:"previewUrl"`
	OriginalURL *string    `json:"originalUrl"`
}

// CreateProfileUseCase is the domain call the profile creation handler depends on.
type CreateProfileUseCase func(ctx context.Context, params usecases.CreateProfileParams) (usecases.CreateProfileResult, error)

// CreateProfileDependencies carries the use case recording the profile and the query re-reading it.
type CreateProfileDependencies struct {
	CreateProfile  CreateProfileUseCase
	GetProfileByID GetProfileByIDUseCase
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

		profileInput, err := input.ValidateProfileInput(input.ProfileInput{
			FirstName:       request.FirstName,
			LastName:        request.LastName,
			PictureUploadID: request.PictureUploadID,
		})
		if err != nil {
			writeCreateProfileFailure(w, err)
			return
		}

		pictureUploadID, err := parsePictureUploadID(profileInput.PictureUploadID)
		if err != nil {
			writeCreateProfileFailure(w, err)
			return
		}

		result, err := d.CreateProfile(r.Context(), usecases.CreateProfileParams{
			UserID:          userID,
			FirstName:       profileInput.FirstName,
			LastName:        profileInput.LastName,
			PictureUploadID: pictureUploadID,
		})
		if err != nil {
			writeCreateProfileFailure(w, err)
			return
		}

		readResult, err := d.GetProfileByID(r.Context(), queries.GetProfileByIDParams{UserID: result.Profile.ID})
		if err != nil {
			writeCreateProfileFailure(w, err)
			return
		}

		handlers.WriteJSON(w, http.StatusCreated, buildCreateProfileResponse(readResult.Profile))
	}
}

func buildCreateProfileResponse(profile models.ProfileReadModel) createProfileResponse {
	return createProfileResponse{
		ID:        profile.ID,
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
		Picture: createProfilePictureResponse{
			UploadID:    profile.Picture.UploadID,
			PreviewURL:  profile.Picture.PreviewURL,
			OriginalURL: profile.Picture.OriginalURL,
		},
	}
}

var createProfileRules = []handlers.RejectionRule{
	handlers.RejectWhen[profileerrors.ErrProfileAlreadyExists](
		handlers.Rejection{Status: http.StatusConflict, Code: handlers.ErrorCodeProfileExists}),
}

func writeCreateProfileFailure(w http.ResponseWriter, err error) {
	handlers.WriteRejection(w, handlers.WriteRejectionParams{
		Err: err,
		RuleSets: [][]handlers.RejectionRule{
			handlers.RequestDecodingRules, profileValidationRules, profilePictureRules, createProfileRules,
		},
		UnexpectedFailureText: unexpectedProfileCreationFailureText,
	})
}
