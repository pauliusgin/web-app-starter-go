package messaging

import (
	"context"
	"log/slog"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/usecases"
	"github.com/pauliusgin/web_app_starter-be/internal/events/auth"
)

// SendPasswordResetEmail is the domain call the password reset email handler depends on.
type SendPasswordResetEmail func(
	ctx context.Context,
	params usecases.SendPasswordResetEmailParams,
) error

// PasswordResetEmailDependencies carries the use case mailing the link and the logger skipped challenges are noted in.
type PasswordResetEmailDependencies struct {
	SendPasswordResetEmail SendPasswordResetEmail
	Logger                 *slog.Logger
}

// PasswordResetEmailOnChallenge returns the handler mailing each new password reset link, or nil.
func PasswordResetEmailOnChallenge(
	d PasswordResetEmailDependencies,
) func(context.Context, auth.AuthChallengeGeneratedData) error {
	if d.SendPasswordResetEmail == nil {
		return nil
	}

	logger := d.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return func(ctx context.Context, payload auth.AuthChallengeGeneratedData) error {
		if challenge.Purpose(payload.Purpose) != challenge.PurposePasswordReset {
			logger.Debug("auth challenge ignored by the password reset email",
				"challengeId", payload.ChallengeID,
				"purpose", payload.Purpose)

			return nil
		}

		return d.SendPasswordResetEmail(ctx, usecases.SendPasswordResetEmailParams{
			RecipientID:    payload.UserID,
			ChallengeID:    payload.ChallengeID,
			RecipientEmail: payload.Target,
			Token:          payload.Token,
		})
	}
}
