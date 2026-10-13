// Package email declares the outbound port the domain sends transactional email through.
package email

import "context"

// VerificationEmailPayload is the recipient and the link a verification email carries.
type VerificationEmailPayload struct {
	RecipientEmail        string
	VerificationLinkParam string
	VerificationLink      string
}

// Response is what the email provider answered with when it accepted a message.
type Response struct {
	ProviderMessageID string
}

// Gateway is the outbound email port the domain calls and an email provider adapter implements.
type Gateway interface {
	SendVerificationEmail(ctx context.Context, p VerificationEmailPayload) (Response, error)
}
