package users

import (
	"context"
	"log/slog"
	"net/http"

	readmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/models"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/queries"
)

type getUserByIDResponse struct {
	readmodels.UserReadModel
}

// GetUserByIDUseCase is the read-side lookup the current-user handler depends on.
type GetUserByIDUseCase func(ctx context.Context, params queries.GetUserByIDParams) (queries.GetUserByIDResult, error)

// GetUserByIDDependencies carries the query reading the caller's user read model.
type GetUserByIDDependencies struct {
	GetUserByID GetUserByIDUseCase
}

// GetUserByID returns a handler answering with the caller's own user read model.
func GetUserByID(d GetUserByIDDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			slog.Error(handlers.RouteNotProtectedText, "path", r.URL.Path)
			handlers.WriteJSON(w, http.StatusInternalServerError,
				handlers.ErrorResponse{Code: handlers.ErrorCodeInternal, Message: unexpectedUserLookupFailureText})

			return
		}

		result, err := d.GetUserByID(r.Context(), queries.GetUserByIDParams{UserID: userID})
		if err != nil {
			writeGetUserByIDFailure(w, err)
			return
		}

		handlers.WriteJSON(w, http.StatusOK, buildGetUserByIDResponse(result.User))
	}
}

var getUserByIDRules = []handlers.RejectionRule{
	handlers.RejectWhen[usererrors.ErrUserNotFound](handlers.Rejection{
		Status:  http.StatusNotFound,
		Code:    handlers.ErrorCodeUserNotFound,
		Message: userNotFoundText,
	}),
}

func buildGetUserByIDResponse(user readmodels.UserReadModel) getUserByIDResponse {
	return getUserByIDResponse{UserReadModel: user}
}

func writeGetUserByIDFailure(w http.ResponseWriter, err error) {
	handlers.WriteRejection(w, handlers.WriteRejectionParams{
		Err:                   err,
		RuleSets:              [][]handlers.RejectionRule{getUserByIDRules},
		UnexpectedFailureText: unexpectedUserLookupFailureText,
	})
}
