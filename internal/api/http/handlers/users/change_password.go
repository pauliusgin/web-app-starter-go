package users

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/input"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// ChangePasswordUseCase is the domain call the password change handler depends on.
type ChangePasswordUseCase func(ctx context.Context, params usecases.ChangePasswordParams) error

// ChangePasswordDependencies carries the use case swapping in the caller's new password.
type ChangePasswordDependencies struct {
	ChangePassword ChangePasswordUseCase
}

// ChangePassword returns a handler replacing the authenticated user's password once the current one matches.
func ChangePassword(d ChangePasswordDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			slog.Error(handlers.RouteNotProtectedText, "path", r.URL.Path)
			handlers.WriteJSON(w, http.StatusInternalServerError,
				handlers.ErrorResponse{Code: handlers.ErrorCodeInternal, Message: unexpectedPasswordChangeFailureText})

			return
		}

		request, err := handlers.DecodeRequest[changePasswordRequest](w, r)
		if err != nil {
			writeChangePasswordFailure(w, err)
			return
		}

		changePasswordInput, err := input.ValidateChangePasswordInput(input.ChangePasswordInput{
			CurrentPassword: request.CurrentPassword,
			NewPassword:     request.NewPassword,
		})
		if err != nil {
			writeChangePasswordFailure(w, err)
			return
		}

		err = d.ChangePassword(r.Context(), usecases.ChangePasswordParams{
			UserID:          userID,
			CurrentPassword: changePasswordInput.CurrentPassword,
			NewPassword:     changePasswordInput.NewPassword,
		})
		if err != nil {
			writeChangePasswordFailure(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

var changePasswordRules = []handlers.RejectionRule{
	handlers.RejectWhen[usererrors.ErrPasswordUnchanged](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodePasswordUnchanged}),
	handlers.RejectWhen[usererrors.ErrCurrentPasswordIncorrect](
		handlers.Rejection{Status: http.StatusForbidden, Code: handlers.ErrorCodeCurrentPasswordIncorrect}),
	handlers.RejectWhen[usererrors.ErrPasswordNotSet](
		handlers.Rejection{Status: http.StatusConflict, Code: handlers.ErrorCodePasswordNotSet}),
}

func writeChangePasswordFailure(w http.ResponseWriter, err error) {
	handlers.WriteRejection(w, handlers.WriteRejectionParams{
		Err:                   err,
		RuleSets:              [][]handlers.RejectionRule{handlers.RequestDecodingRules, passwordValidationRules, changePasswordRules},
		UnexpectedFailureText: unexpectedPasswordChangeFailureText,
	})
}
