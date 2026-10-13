package brevo

import "fmt"

const gatewayNotWiredText = "expected a gateway built by NewEmailGateway"
const apiKeyMissingText = "expected the account's API key, and not the placeholder config leaves when the variable is unset"
const senderEmailMissingText = "expected the verified address emails are sent from, and not the placeholder config leaves when the variable is unset"
const verificationTemplateIDInvalidText = "expected the positive id of the stored template the verification email is rendered from"
const sendRejectedText = "brevo refused the verification email"
const sendUnansweredText = "brevo never answered, so whether the verification email was accepted is unknown"

// ErrGatewayNotWired reports a send through the zero value of EmailGateway.
type ErrGatewayNotWired struct {
	RecipientEmail string
}

func (e ErrGatewayNotWired) Error() string {
	return fmt.Sprintf("cannot send to %q: %s", e.RecipientEmail, gatewayNotWiredText)
}

// ErrApiKeyMissing reports a gateway built without a Brevo API key.
type ErrApiKeyMissing struct{}

func (e ErrApiKeyMissing) Error() string {
	return fmt.Sprintf("missing brevo api key: %s", apiKeyMissingText)
}

// ErrSenderEmailMissing reports a gateway built without a sender address.
type ErrSenderEmailMissing struct{}

func (e ErrSenderEmailMissing) Error() string {
	return fmt.Sprintf("missing brevo sender email: %s", senderEmailMissingText)
}

// ErrVerificationTemplateIDInvalid reports a verification template ID that is not positive.
type ErrVerificationTemplateIDInvalid struct {
	TemplateID int
}

func (e ErrVerificationTemplateIDInvalid) Error() string {
	return fmt.Sprintf("invalid brevo verification template id %d: %s",
		e.TemplateID, verificationTemplateIDInvalidText)
}

// ErrSendRejected reports a verification email Brevo answered for and refused.
type ErrSendRejected struct {
	RecipientEmail string
	StatusCode     int
	BodySnippet    string
	Cause          error
}

func (e ErrSendRejected) Error() string {
	return fmt.Sprintf("%s to %q: status %d: %s: body %q",
		sendRejectedText, e.RecipientEmail, e.StatusCode, e.Cause, e.BodySnippet)
}

func (e ErrSendRejected) Unwrap() error {
	return e.Cause
}

// ErrSendUnanswered reports a send that got no HTTP response, so its outcome is unknown.
type ErrSendUnanswered struct {
	RecipientEmail string
	Cause          error
}

func (e ErrSendUnanswered) Error() string {
	return fmt.Sprintf("%s to %q: %s", sendUnansweredText, e.RecipientEmail, e.Cause)
}

func (e ErrSendUnanswered) Unwrap() error {
	return e.Cause
}
