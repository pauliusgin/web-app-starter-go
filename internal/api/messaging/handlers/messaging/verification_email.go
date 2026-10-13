// Package messaging turns consumed events into messaging domain calls.
package messaging

import (
	"context"
	"log/slog"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/usecases"
	"github.com/pauliusgin/web_app_starter-be/internal/events/auth"
)

// SendVerificationEmail is the domain call the verification email handler depends on.
type SendVerificationEmail func(
	ctx context.Context,
	params usecases.SendVerificationEmailParams,
) error

// VerificationEmailDependencies carries the collaborators the verification email handler needs.
type VerificationEmailDependencies struct {
	SendVerificationEmail SendVerificationEmail
	Logger                *slog.Logger
}

// verificationMessageTypeByPurpose lists the challenge purposes answered with an emailed link.
var verificationMessageTypeByPurpose = map[challenge.Purpose]models.MessageType{
	challenge.PurposeEmailVerification: models.EmailVerification,
}

// VerificationEmailOnChallenge returns the handler mailing each new verification link, or nil.
func VerificationEmailOnChallenge(
	d VerificationEmailDependencies,
) func(context.Context, auth.AuthChallengeGeneratedData) error {
	if d.SendVerificationEmail == nil {
		return nil
	}

	logger := d.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return func(ctx context.Context, payload auth.AuthChallengeGeneratedData) error {
		messageType, mailed := verificationMessageTypeByPurpose[challenge.Purpose(payload.Purpose)]
		if !mailed {
			logger.Debug("auth challenge ignored",
				"challenge_id", payload.ChallengeID,
				"purpose", payload.Purpose)

			return nil
		}

		return d.SendVerificationEmail(ctx, usecases.SendVerificationEmailParams{
			RecipientID:    payload.UserID,
			ChallengeID:    payload.ChallengeID,
			RecipientEmail: payload.Target,
			Token:          payload.Token,
			Type:           messageType,
		})
	}
}
