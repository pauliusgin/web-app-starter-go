// Package messages holds the HTTP handlers that ask the server to send a message to the caller.
package messages

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

// ResendEmailVerificationUseCase is the domain call the verification email resend handler depends on.
type ResendEmailVerificationUseCase func(ctx context.Context, params usecases.ResendEmailVerificationParams) error

// ResendEmailVerificationDependencies carries the use case mailing the caller a fresh verification link.
type ResendEmailVerificationDependencies struct {
	ResendEmailVerification ResendEmailVerificationUseCase
}

// ResendEmailVerification returns a handler mailing a fresh verification link to the caller's unverified email.
func ResendEmailVerification(d ResendEmailVerificationDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			slog.Error(handlers.RouteNotProtectedText, "path", r.URL.Path)
			handlers.WriteJSON(w, http.StatusInternalServerError,
				handlers.ErrorResponse{Code: handlers.ErrorCodeInternal, Message: unexpectedVerificationEmailResendFailureText})

			return
		}

		err := d.ResendEmailVerification(r.Context(), usecases.ResendEmailVerificationParams{UserID: userID})
		if err != nil {
			writeResendEmailVerificationFailure(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

var resendEmailVerificationRules = []handlers.RejectionRule{
	handlers.RejectWhen[usererrors.ErrEmailAlreadyVerified](
		handlers.Rejection{Status: http.StatusConflict, Code: handlers.ErrorCodeEmailAlreadyVerified}),
	handlers.RejectWhen[usererrors.ErrUserNotFound](handlers.Rejection{
		Status:  http.StatusNotFound,
		Code:    handlers.ErrorCodeUserNotFound,
		Message: userNotFoundText,
	}),
}

func writeResendEmailVerificationFailure(w http.ResponseWriter, err error) {
	handlers.WriteRetryAfterHeader(w, err)
	handlers.WriteRejection(w, handlers.WriteRejectionParams{
		Err:                   err,
		RuleSets:              [][]handlers.RejectionRule{resendEmailVerificationRules, handlers.VerificationEmailRateLimitRules},
		UnexpectedFailureText: unexpectedVerificationEmailResendFailureText,
	})
}
