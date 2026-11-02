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

type addGoogleAccountRequest struct {
	IDToken string `json:"idToken"`
}

// AddGoogleAccountUseCase is the domain call the google account addition handler depends on.
type AddGoogleAccountUseCase func(ctx context.Context, params usecases.AddGoogleAccountParams) error

// AddGoogleAccountDependencies carries the use case connecting a google account to the caller's password account.
type AddGoogleAccountDependencies struct {
	AddGoogleAccount AddGoogleAccountUseCase
}

// AddGoogleAccount returns a handler connecting the google account holding the authenticated user's email.
func AddGoogleAccount(d AddGoogleAccountDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			slog.Error(handlers.RouteNotProtectedText, "path", r.URL.Path)
			handlers.WriteJSON(w, http.StatusInternalServerError,
				handlers.ErrorResponse{Code: handlers.ErrorCodeInternal, Message: unexpectedGoogleAccountAdditionFailureText})
			return
		}

		request, err := handlers.DecodeRequest[addGoogleAccountRequest](w, r)
		if err != nil {
			writeAddGoogleAccountFailure(w, err)
			return
		}

		googleIDTokenInput, err := input.ValidateGoogleIDTokenInput(input.GoogleIDTokenInput{IDToken: request.IDToken})
		if err != nil {
			writeAddGoogleAccountFailure(w, err)
			return
		}

		err = d.AddGoogleAccount(r.Context(), usecases.AddGoogleAccountParams{
			UserID:  userID,
			IDToken: googleIDTokenInput.IDToken,
		})
		if err != nil {
			writeAddGoogleAccountFailure(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

var addGoogleAccountRules = []handlers.RejectionRule{
	handlers.RejectWhen[usererrors.ErrGoogleAccountAlreadyConnected](
		handlers.Rejection{Status: http.StatusConflict, Code: handlers.ErrorCodeGoogleAccountAlreadyConnected}),
	handlers.RejectWhen[usererrors.ErrGoogleEmailMismatch](
		handlers.Rejection{Status: http.StatusConflict, Code: handlers.ErrorCodeGoogleEmailMismatch}),
	handlers.RejectWhen[usererrors.ErrGoogleAccountTaken](
		handlers.Rejection{Status: http.StatusConflict, Code: handlers.ErrorCodeGoogleAccountTaken}),
}

func writeAddGoogleAccountFailure(w http.ResponseWriter, err error) {
	handlers.WriteRejection(w, handlers.WriteRejectionParams{
		Err:                   err,
		RuleSets:              [][]handlers.RejectionRule{handlers.RequestDecodingRules, googleIdentityRules, addGoogleAccountRules},
		UnexpectedFailureText: unexpectedGoogleAccountAdditionFailureText,
	})
}
