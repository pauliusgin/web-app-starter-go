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

type requestEmailChangeRequest struct {
	NewEmail string `json:"newEmail"`
}

// RequestEmailChangeUseCase is the domain call the email change request handler depends on.
type RequestEmailChangeUseCase func(ctx context.Context, params usecases.RequestEmailChangeParams) error

// RequestEmailChangeDependencies carries the collaborator the email change request handler orchestrates.
type RequestEmailChangeDependencies struct {
	RequestEmailChange RequestEmailChangeUseCase
}

// RequestEmailChange returns a handler mailing a link to the new address the authenticated user asked for.
func RequestEmailChange(d RequestEmailChangeDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			slog.Error(handlers.RouteNotProtectedText, "path", r.URL.Path)
			handlers.WriteJSON(w, http.StatusInternalServerError,
				handlers.ErrorResponse{Code: handlers.ErrorCodeInternal, Message: unexpectedEmailChangeRequestFailureText})

			return
		}

		request, err := handlers.DecodeRequest[requestEmailChangeRequest](w, r)
		if err != nil {
			writeRequestEmailChangeFailure(w, err)
			return
		}

		requestEmailChangeInput := input.RequestEmailChangeInput{
			NewEmail: normalizeEmail(request.NewEmail),
		}
		if err := input.ValidateRequestEmailChangeInput(requestEmailChangeInput); err != nil {
			writeRequestEmailChangeFailure(w, err)
			return
		}

		err = d.RequestEmailChange(r.Context(), usecases.RequestEmailChangeParams{
			UserID:   userID,
			NewEmail: requestEmailChangeInput.NewEmail,
		})
		if err != nil {
			writeRequestEmailChangeFailure(w, err)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

var requestEmailChangeRules = []handlers.RejectionRule{
	handlers.RejectWhen[usererrors.ErrEmailUnchanged](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeEmailUnchanged}),
	handlers.RejectWhen[usererrors.ErrEmailTaken](
		handlers.Rejection{Status: http.StatusConflict, Code: handlers.ErrorCodeEmailTaken}),
}

func writeRequestEmailChangeFailure(w http.ResponseWriter, err error) {
	handlers.WriteRejection(w, handlers.WriteRejectionParams{
		Err:                   err,
		RuleSets:              [][]handlers.RejectionRule{handlers.RequestDecodingRules, emailValidationRules, requestEmailChangeRules},
		UnexpectedFailureText: unexpectedEmailChangeRequestFailureText,
	})
}
