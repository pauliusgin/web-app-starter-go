package users

import (
	"context"
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/input"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

// ResetPasswordUseCase is the domain call the password reset handler depends on.
type ResetPasswordUseCase func(ctx context.Context, params usecases.ResetPasswordParams) error

// ResetPasswordDependencies carries the use case setting a new password through a reset link.
type ResetPasswordDependencies struct {
	ResetPassword ResetPasswordUseCase
}

// ResetPassword returns a handler setting the new password of the account an emailed reset link was issued to.
func ResetPassword(d ResetPasswordDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request, err := handlers.DecodeRequest[resetPasswordRequest](w, r)
		if err != nil {
			writeResetPasswordFailure(w, err)
			return
		}

		resetPasswordInput, err := input.ValidateResetPasswordInput(input.ResetPasswordInput{
			Token:       request.Token,
			NewPassword: request.NewPassword,
		})
		if err != nil {
			writeResetPasswordFailure(w, err)
			return
		}

		err = d.ResetPassword(r.Context(), usecases.ResetPasswordParams{
			Token:       resetPasswordInput.Token,
			NewPassword: resetPasswordInput.NewPassword,
		})
		if err != nil {
			writeResetPasswordFailure(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

var resetPasswordRules = []handlers.RejectionRule{
	handlers.RejectWhen[primitive.ErrPasswordResetTokenMissing](handlers.Rejection{
		Status:  http.StatusBadRequest,
		Code:    handlers.ErrorCodePasswordResetTokenMissing,
		Message: passwordResetTokenRequiredText,
	}),
	handlers.RejectWhen[usererrors.ErrPasswordResetLinkInvalid](handlers.Rejection{
		Status:  http.StatusBadRequest,
		Code:    handlers.ErrorCodePasswordResetLinkInvalid,
		Message: invalidPasswordResetLinkText,
	}),
}

func writeResetPasswordFailure(w http.ResponseWriter, err error) {
	handlers.WriteRejection(w, handlers.WriteRejectionParams{
		Err:                   err,
		RuleSets:              [][]handlers.RejectionRule{handlers.RequestDecodingRules, resetPasswordRules, passwordValidationRules},
		UnexpectedFailureText: unexpectedPasswordResetFailureText,
	})
}
