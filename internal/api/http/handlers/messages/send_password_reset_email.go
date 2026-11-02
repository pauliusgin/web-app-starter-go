package messages

import (
	"context"
	"net/http"
	"time"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/input"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

// passwordResetResponseFloor outlasts the slowest path through RequestPasswordReset, so a known email
// answers no later than an unknown one.
const passwordResetResponseFloor = time.Second

type sendPasswordResetEmailRequest struct {
	Email string `json:"email"`
}

// RequestPasswordResetUseCase is the domain call the password reset email handler depends on.
type RequestPasswordResetUseCase func(ctx context.Context, params usecases.RequestPasswordResetParams) error

// SendPasswordResetEmailDependencies carries the use case mailing the reset link and the clock padding the answer.
type SendPasswordResetEmailDependencies struct {
	RequestPasswordReset RequestPasswordResetUseCase
	Clock                handlers.ResponseClock
}

// SendPasswordResetEmail returns a handler mailing a password reset link to the email's password account. It
// answers the same, no sooner than passwordResetResponseFloor, whether or not the email has an account.
func SendPasswordResetEmail(d SendPasswordResetEmailDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startedAt := d.Clock.Now()

		request, err := handlers.DecodeRequest[sendPasswordResetEmailRequest](w, r)
		if err != nil {
			writeSendPasswordResetEmailFailure(w, err)
			return
		}

		passwordResetEmailInput, err := input.ValidatePasswordResetEmailInput(
			input.PasswordResetEmailInput{Email: request.Email})
		if err != nil {
			writeSendPasswordResetEmailFailure(w, err)
			return
		}

		err = d.RequestPasswordReset(r.Context(), usecases.RequestPasswordResetParams{
			Email: passwordResetEmailInput.Email,
		})

		handlers.WaitForResponseFloor(r.Context(), handlers.WaitForResponseFloorParams{
			StartedAt: startedAt,
			Floor:     passwordResetResponseFloor,
			Clock:     d.Clock,
		})

		if err != nil {
			writeSendPasswordResetEmailFailure(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

var passwordResetEmailValidationRules = []handlers.RejectionRule{
	handlers.RejectWhen[primitive.ErrEmailInvalid](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeEmailInvalid}),
}

func writeSendPasswordResetEmailFailure(w http.ResponseWriter, err error) {
	handlers.WriteRejection(w, handlers.WriteRejectionParams{
		Err:                   err,
		RuleSets:              [][]handlers.RejectionRule{handlers.RequestDecodingRules, passwordResetEmailValidationRules},
		UnexpectedFailureText: unexpectedPasswordResetEmailFailureText,
	})
}
