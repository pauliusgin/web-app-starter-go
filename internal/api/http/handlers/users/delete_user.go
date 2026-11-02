package users

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

// DeleteUserUseCase is the domain call the deletion handler depends on.
type DeleteUserUseCase func(ctx context.Context, params usecases.DeleteUserParams) error

// DeleteUserDependencies carries the use case soft-deleting the caller's account.
type DeleteUserDependencies struct {
	DeleteUser DeleteUserUseCase
}

// DeleteUser returns a handler soft-deleting the caller's own account.
func DeleteUser(d DeleteUserDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			slog.Error(handlers.RouteNotProtectedText, "path", r.URL.Path)
			handlers.WriteJSON(w, http.StatusInternalServerError,
				handlers.ErrorResponse{Code: handlers.ErrorCodeInternal, Message: unexpectedDeletionFailureText})

			return
		}

		if err := d.DeleteUser(r.Context(), usecases.DeleteUserParams{UserID: userID}); err != nil {
			writeDeleteUserFailure(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

var deleteUserRules = []handlers.RejectionRule{
	handlers.RejectWhen[usererrors.ErrUserNotFound](handlers.Rejection{
		Status:  http.StatusNotFound,
		Code:    handlers.ErrorCodeUserNotFound,
		Message: userNotFoundText,
	}),
}

func writeDeleteUserFailure(w http.ResponseWriter, err error) {
	handlers.WriteRejection(w, handlers.WriteRejectionParams{
		Err:                   err,
		RuleSets:              [][]handlers.RejectionRule{deleteUserRules},
		UnexpectedFailureText: unexpectedDeletionFailureText,
	})
}
