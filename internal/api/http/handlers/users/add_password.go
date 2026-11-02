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

type addPasswordRequest struct {
	Password string `json:"password"`
}

// AddPasswordUseCase is the domain call the password addition handler depends on.
type AddPasswordUseCase func(ctx context.Context, params usecases.AddPasswordParams) error

// AddPasswordDependencies carries the use case adding a password to the caller's google-only account.
type AddPasswordDependencies struct {
	AddPassword AddPasswordUseCase
}

// AddPassword returns a handler adding a password to the authenticated google-only user, so it signs in with either.
func AddPassword(d AddPasswordDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			slog.Error(handlers.RouteNotProtectedText, "path", r.URL.Path)
			handlers.WriteJSON(w, http.StatusInternalServerError,
				handlers.ErrorResponse{Code: handlers.ErrorCodeInternal, Message: unexpectedPasswordAdditionFailureText})
			return
		}

		request, err := handlers.DecodeRequest[addPasswordRequest](w, r)
		if err != nil {
			writeAddPasswordFailure(w, err)
			return
		}

		addPasswordInput, err := input.ValidateAddPasswordInput(input.AddPasswordInput{Password: request.Password})
		if err != nil {
			writeAddPasswordFailure(w, err)
			return
		}

		err = d.AddPassword(r.Context(), usecases.AddPasswordParams{
			UserID:   userID,
			Password: addPasswordInput.Password,
		})
		if err != nil {
			writeAddPasswordFailure(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

var addPasswordRules = []handlers.RejectionRule{
	handlers.RejectWhen[usererrors.ErrPasswordAlreadySet](
		handlers.Rejection{Status: http.StatusConflict, Code: handlers.ErrorCodePasswordAlreadySet}),
	handlers.RejectWhen[usererrors.ErrGoogleAccountNotConnected](
		handlers.Rejection{Status: http.StatusConflict, Code: handlers.ErrorCodeGoogleAccountNotConnected}),
}

func writeAddPasswordFailure(w http.ResponseWriter, err error) {
	handlers.WriteRejection(w, handlers.WriteRejectionParams{
		Err:                   err,
		RuleSets:              [][]handlers.RejectionRule{handlers.RequestDecodingRules, passwordValidationRules, addPasswordRules},
		UnexpectedFailureText: unexpectedPasswordAdditionFailureText,
	})
}
