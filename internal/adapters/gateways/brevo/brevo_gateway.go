// Package brevo sends the domain's transactional email through Brevo's HTTP API.
package brevo

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	brevosdk "github.com/getbrevo/brevo-go/lib"

	email "github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/gateways/email"
)

const apiKeyHeader = "api-key"

const sendRequestTimeout = 10 * time.Second
const httpClientTimeout = 15 * time.Second

const maxBodySnippetLength = 512
const bodySnippetTruncatedMarker = "…[truncated]"

var _ email.Gateway = EmailGateway{}

// EmailGateway sends verification email through one Brevo account's transactional API.
type EmailGateway struct {
	client                 *brevosdk.APIClient
	senderEmail            string
	verificationTemplateID int64
}

// EmailGatewayParams carries the Brevo account details one gateway sends on behalf of.
type EmailGatewayParams struct {
	ApiKey                 string
	SenderEmail            string
	VerificationTemplateID int
}

// NewEmailGateway validates the account details and builds the one client the process reuses.
func NewEmailGateway(params EmailGatewayParams) (EmailGateway, error) {
	if err := ensureAccountConfigured(params); err != nil {
		return EmailGateway{}, err
	}

	configuration := brevosdk.NewConfiguration()
	configuration.AddDefaultHeader(apiKeyHeader, params.ApiKey)
	configuration.HTTPClient = &http.Client{Timeout: httpClientTimeout}

	return EmailGateway{
		client:                 brevosdk.NewAPIClient(configuration),
		senderEmail:            params.SenderEmail,
		verificationTemplateID: int64(params.VerificationTemplateID),
	}, nil
}

// SendVerificationEmail renders the account's verification template around the link and sends it.
func (g EmailGateway) SendVerificationEmail(
	ctx context.Context,
	p email.VerificationEmailPayload,
) (email.Response, error) {
	if g.client == nil {
		return email.Response{}, ErrGatewayNotWired{RecipientEmail: p.RecipientEmail}
	}

	sendCtx, cancelSend := context.WithTimeout(ctx, sendRequestTimeout)
	defer cancelSend()

	created, response, err := g.client.TransactionalEmailsApi.SendTransacEmail(
		sendCtx,
		g.buildVerificationEmail(p),
	)
	if err != nil {
		return email.Response{}, describeSendFailure(p.RecipientEmail, response, err)
	}

	return email.Response{ProviderMessageID: created.MessageId}, nil
}

func (g EmailGateway) buildVerificationEmail(p email.VerificationEmailPayload) brevosdk.SendSmtpEmail {
	return brevosdk.SendSmtpEmail{
		Sender:     &brevosdk.SendSmtpEmailSender{Email: g.senderEmail},
		To:         []brevosdk.SendSmtpEmailTo{{Email: p.RecipientEmail}},
		TemplateId: g.verificationTemplateID,
		Params: map[string]any{
			p.VerificationLinkParam: p.VerificationLink,
		},
	}
}

func describeSendFailure(recipientEmail string, response *http.Response, cause error) error {
	if response == nil {
		return ErrSendUnanswered{RecipientEmail: recipientEmail, Cause: cause}
	}

	bodySnippet := ""

	var refusal brevosdk.GenericSwaggerError
	if errors.As(cause, &refusal) {
		bodySnippet = truncateBodySnippet(refusal.Body())
	}

	return ErrSendRejected{
		RecipientEmail: recipientEmail,
		StatusCode:     response.StatusCode,
		BodySnippet:    bodySnippet,
		Cause:          cause,
	}
}

func truncateBodySnippet(body []byte) string {
	if len(body) <= maxBodySnippetLength {
		return string(body)
	}

	return strings.ToValidUTF8(string(body[:maxBodySnippetLength]), "") + bodySnippetTruncatedMarker
}

func ensureAccountConfigured(params EmailGatewayParams) error {
	if params.ApiKey == "" {
		return ErrApiKeyMissing{}
	}
	if params.SenderEmail == "" {
		return ErrSenderEmailMissing{}
	}
	if params.VerificationTemplateID <= 0 {
		return ErrVerificationTemplateIDInvalid{TemplateID: params.VerificationTemplateID}
	}

	return nil
}
