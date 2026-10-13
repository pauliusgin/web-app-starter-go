package users

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/tokens"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	messaging "github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/usecases"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	users "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

// VerifyEmailUseCase is the domain call the verification handler depends on.
type VerifyEmailUseCase func(
	ctx context.Context,
	params users.VerifyEmailParams,
) error

// VerifyEmailDependencies carries the collaborator the verification handler orchestrates.
type VerifyEmailDependencies struct {
	VerifyEmail VerifyEmailUseCase
}

// VerifyEmail returns a handler redeeming an email verification link for the authenticated user:
// their current email is marked verified, or the address the link proved becomes their email.
func VerifyEmail(d VerifyEmailDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			slog.Error(handlers.RouteNotProtectedText, "path", r.URL.Path)
			handlers.WriteJSON(w, http.StatusInternalServerError,
				handlers.ErrorResponse{Code: handlers.ErrorCodeInternal, Message: unexpectedVerificationFailureText})

			return
		}

		verificationToken := r.URL.Query().Get(messaging.VerificationTokenQueryParam)
		if verificationToken == "" {
			writeVerifyEmailFailure(w, tokens.ErrVerificationTokenAbsent{})
			return
		}

		if err := d.VerifyEmail(r.Context(), users.VerifyEmailParams{
			VerificationToken: verificationToken,
			UserID:            userID,
		}); err != nil {
			writeVerifyEmailFailure(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

var verifyEmailRules = []handlers.RejectionRule{
	handlers.RejectWhen[tokens.ErrVerificationTokenAbsent](handlers.Rejection{
		Status:  http.StatusBadRequest,
		Code:    handlers.ErrorCodeVerificationTokenMissing,
		Message: tokens.VerificationTokenRequiredText,
	}),
	handlers.RejectWhen[usererrors.ErrVerificationLinkInvalid](handlers.Rejection{
		Status:  http.StatusBadRequest,
		Code:    handlers.ErrorCodeVerificationLinkInvalid,
		Message: invalidVerificationLinkText,
	}),
	handlers.RejectWhen[challenge.ErrChallengePurposeMismatch](handlers.Rejection{
		Status:  http.StatusBadRequest,
		Code:    handlers.ErrorCodeVerificationLinkInvalid,
		Message: invalidVerificationLinkText,
	}),
	handlers.RejectWhen[challenge.ErrChallengeOwnedByAnotherUser](handlers.Rejection{
		Status:  http.StatusForbidden,
		Code:    handlers.ErrorCodeVerificationLinkNotOwned,
		Message: verificationLinkNotYoursText,
	}),
	handlers.RejectWhen[usererrors.ErrEmailTaken](
		handlers.Rejection{Status: http.StatusConflict, Code: handlers.ErrorCodeEmailTaken}),
}

func writeVerifyEmailFailure(w http.ResponseWriter, err error) {
	handlers.WriteRejection(w, handlers.WriteRejectionParams{
		Err:                   err,
		RuleSets:              [][]handlers.RejectionRule{verifyEmailRules},
		UnexpectedFailureText: unexpectedVerificationFailureText,
		RejectedLogText:       verificationRejectedText,
	})
}
