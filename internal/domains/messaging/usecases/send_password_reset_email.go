package usecases

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/gateways/email"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

const passwordResetLinkParam = "passwordResetLink"

// SendPasswordResetEmailParams is the recipient, the password reset challenge and its token.
type SendPasswordResetEmailParams struct {
	RecipientID    uuid.UUID
	ChallengeID    uuid.UUID
	RecipientEmail string
	Token          string
}

// SendPasswordResetEmailDependencies carries what sending, logging and announcing an email needs.
type SendPasswordResetEmailDependencies struct {
	Gateway           email.Gateway
	MessageRepository repositories.MessageWriteRepository
	EventDispatcher   events.Dispatcher
	Logger            *slog.Logger
	FrontendBaseURL   string
}

// SendPasswordResetEmail builds the password reset link, hands the message to the email gateway, then
// logs and announces the outcome. Logging failures are not returned: the email has already left.
func SendPasswordResetEmail(
	ctx context.Context,
	d SendPasswordResetEmailDependencies,
	params SendPasswordResetEmailParams,
) error {
	err := ensureTokenMessageRecipient(tokenMessageRecipient{
		RecipientID:    params.RecipientID,
		RecipientEmail: params.RecipientEmail,
		Token:          params.Token,
	})
	if err != nil {
		return err
	}

	passwordResetLink, err := buildTokenLink(buildTokenLinkParams{
		FrontendBaseURL: d.FrontendBaseURL,
		Token:           params.Token,
		Type:            models.PasswordReset,
	})
	if err != nil {
		return err
	}

	response, sendErr := d.Gateway.SendPasswordResetEmail(ctx, email.PasswordResetEmailPayload{
		RecipientEmail:         params.RecipientEmail,
		PasswordResetLinkParam: passwordResetLinkParam,
		PasswordResetLink:      passwordResetLink,
	})

	logMessageOutcome(ctx, logMessageOutcomeDependencies{
		MessageRepository: d.MessageRepository,
		EventDispatcher:   d.EventDispatcher,
		Logger:            d.Logger,
	}, buildPasswordResetMessage(buildPasswordResetMessageParams{
		Params:            params,
		SentEmailResponse: response,
		SendErr:           sendErr,
	}))

	if sendErr != nil {
		return e.ErrSendPasswordResetEmailFailed{RecipientEmail: params.RecipientEmail, Cause: sendErr}
	}

	return nil
}

type buildPasswordResetMessageParams struct {
	Params            SendPasswordResetEmailParams
	SentEmailResponse email.Response
	SendErr           error
}

func buildPasswordResetMessage(p buildPasswordResetMessageParams) models.NewMessage {
	data, _ := json.Marshal(challengeMessageData{ChallengeID: p.Params.ChallengeID})

	return models.New(models.NewMessageParams{
		RecipientID:    p.Params.RecipientID,
		RecipientEmail: p.Params.RecipientEmail,
		Data:           data,
		Type:           models.PasswordReset,
		Method:         models.MethodEmail,
		Status:         messageStatusFor(p.SendErr),
		ReferenceID:    p.SentEmailResponse.ProviderMessageID,
		Errors:         describeSendErrors(p.SendErr),
	})
}
