// Package usecases holds the messaging use cases, such as sending the verification email.
package usecases

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/url"

	"github.com/google/uuid"

	e "github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/gateways/email"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

// frontendPathByMessageType is where the frontend serves each token link; it must match its routes.
var frontendPathByMessageType = map[models.MessageType]string{
	models.EmailVerification: "/verify-email",
	models.PasswordReset:     "/reset-password",
}

const verificationLinkParam = "verificationLink"

// VerificationTokenQueryParam is the query parameter every token link carries its token in.
const VerificationTokenQueryParam = "token"

// SendVerificationEmailParams is the recipient, the challenge being verified, its token and message type.
type SendVerificationEmailParams struct {
	RecipientID    uuid.UUID
	ChallengeID    uuid.UUID
	RecipientEmail string
	Token          string
	Type           models.MessageType
}

// SendVerificationEmailDependencies carries what sending, logging and announcing an email needs.
type SendVerificationEmailDependencies struct {
	Gateway           email.Gateway
	MessageRepository repositories.MessageWriteRepository
	EventDispatcher   events.Dispatcher
	Logger            *slog.Logger
	FrontendBaseURL   string
}

type challengeMessageData struct {
	ChallengeID uuid.UUID `json:"challengeId"`
}

// SendVerificationEmail builds the verification link, hands the message to the email gateway, then
// logs and announces the outcome. Logging failures are not returned: the email has already left.
func SendVerificationEmail(
	ctx context.Context,
	d SendVerificationEmailDependencies,
	params SendVerificationEmailParams,
) error {

	err := ensureTokenMessageRecipient(tokenMessageRecipient{
		RecipientID:    params.RecipientID,
		RecipientEmail: params.RecipientEmail,
		Token:          params.Token,
	})
	if err != nil {
		return err
	}

	verificationLink, err := buildTokenLink(buildTokenLinkParams{
		FrontendBaseURL: d.FrontendBaseURL,
		Token:           params.Token,
		Type:            params.Type,
	})
	if err != nil {
		return err
	}

	response, sendErr := d.Gateway.SendVerificationEmail(ctx, email.VerificationEmailPayload{
		RecipientEmail:        params.RecipientEmail,
		VerificationLinkParam: verificationLinkParam,
		VerificationLink:      verificationLink,
	})

	logMessageOutcome(ctx, logMessageOutcomeDependencies{
		MessageRepository: d.MessageRepository,
		EventDispatcher:   d.EventDispatcher,
		Logger:            d.Logger,
	}, buildVerificationMessage(buildVerificationMessageParams{
		Params:            params,
		SentEmailResponse: response,
		SendErr:           sendErr,
	}))

	if sendErr != nil {
		return e.ErrSendVerificationEmailFailed{RecipientEmail: params.RecipientEmail, Cause: sendErr}
	}

	return nil
}

type buildVerificationMessageParams struct {
	Params            SendVerificationEmailParams
	SentEmailResponse email.Response
	SendErr           error
}

func buildVerificationMessage(p buildVerificationMessageParams) models.NewMessage {
	data, _ := json.Marshal(challengeMessageData{ChallengeID: p.Params.ChallengeID})

	return models.New(models.NewMessageParams{
		RecipientID:    p.Params.RecipientID,
		RecipientEmail: p.Params.RecipientEmail,
		Data:           data,
		Type:           p.Params.Type,
		Method:         models.MethodEmail,
		Status:         messageStatusFor(p.SendErr),
		ReferenceID:    p.SentEmailResponse.ProviderMessageID,
		Errors:         describeSendErrors(p.SendErr),
	})
}

type tokenMessageRecipient struct {
	RecipientID    uuid.UUID
	RecipientEmail string
	Token          string
}

func ensureTokenMessageRecipient(params tokenMessageRecipient) error {
	if params.RecipientID == uuid.Nil {
		return e.ErrRecipientIDMissing{}
	}
	if params.RecipientEmail == "" {
		return e.ErrRecipientEmailMissing{}
	}
	if params.Token == "" {
		return e.ErrVerificationTokenMissing{}
	}

	return nil
}

type buildTokenLinkParams struct {
	FrontendBaseURL string
	Token           string
	Type            models.MessageType
}

func buildTokenLink(params buildTokenLinkParams) (string, error) {
	frontendPath, supported := frontendPathByMessageType[params.Type]
	if !supported {
		return "", e.ErrVerificationMessageTypeUnsupported{Type: string(params.Type)}
	}
	if params.FrontendBaseURL == "" {
		return "", e.ErrFrontendBaseURLMissing{}
	}

	baseURL, err := url.Parse(params.FrontendBaseURL)
	if err != nil {
		return "", e.ErrFrontendBaseURLUnparseable{
			FrontendBaseURL: params.FrontendBaseURL,
			Cause:           err,
		}
	}
	if baseURL.Scheme == "" || baseURL.Host == "" {
		return "", e.ErrFrontendBaseURLIncomplete{FrontendBaseURL: params.FrontendBaseURL}
	}

	tokenLink := baseURL.JoinPath(frontendPath)

	query := tokenLink.Query()
	query.Set(VerificationTokenQueryParam, params.Token)
	tokenLink.RawQuery = query.Encode()

	return tokenLink.String(), nil
}
