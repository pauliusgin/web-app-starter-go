package users

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	users "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

// UserIDPathValue is the path wildcard naming the user a /users/{userID} route acts on.
const UserIDPathValue = "userID"

// DeleteUserUseCase is the domain call the deletion handler depends on.
type DeleteUserUseCase func(ctx context.Context, params users.DeleteUserParams) error

// DeleteUserDependencies carries the collaborator the deletion handler orchestrates.
type DeleteUserDependencies struct {
	DeleteUser DeleteUserUseCase
}

// DeleteUser returns a handler soft-deleting the account in the path when it is the caller's own.
func DeleteUser(d DeleteUserDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actingUserID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			slog.Error(handlers.RouteNotProtectedText, "path", r.URL.Path)
			handlers.WriteJSON(w, http.StatusInternalServerError,
				handlers.ErrorResponse{Code: handlers.ErrorCodeInternal, Message: unexpectedDeletionFailureText})

			return
		}

		rawTargetUserID := r.PathValue(UserIDPathValue)
		targetUserID, err := uuid.Parse(rawTargetUserID)
		if err != nil {
			writeDeleteUserFailure(w, ErrUserIDMalformed{RawUserID: rawTargetUserID, Cause: err})
			return
		}

		if err := d.DeleteUser(r.Context(), users.DeleteUserParams{
			ActingUserID: actingUserID,
			TargetUserID: targetUserID,
		}); err != nil {
			writeDeleteUserFailure(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

var deleteUserRules = []handlers.RejectionRule{
	handlers.RejectWhen[ErrUserIDMalformed](handlers.Rejection{
		Status:  http.StatusBadRequest,
		Code:    handlers.ErrorCodeUserIDMalformed,
		Message: userIDMalformedText,
	}),
	handlers.RejectWhen[usererrors.ErrUserDeletionForbidden](handlers.Rejection{
		Status:  http.StatusForbidden,
		Code:    handlers.ErrorCodeUserDeletionForbidden,
		Message: deletionNotYoursText,
	}),
	handlers.RejectWhen[usererrors.ErrUserNotFound](handlers.Rejection{
		Status:  http.StatusNotFound,
		Code:    handlers.ErrorCodeUserNotFound,
		Message: deletedUserNotFoundText,
	}),
}

func writeDeleteUserFailure(w http.ResponseWriter, err error) {
	handlers.WriteRejection(w, handlers.WriteRejectionParams{
		Err:                   err,
		RuleSets:              [][]handlers.RejectionRule{deleteUserRules},
		UnexpectedFailureText: unexpectedDeletionFailureText,
		RejectedLogText:       deletionRejectedText,
	})
}
