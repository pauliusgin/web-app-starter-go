// Package errors holds the failures the messaging use cases report.
package errors

import "fmt"

const recipientEmailMissingText = "expected the address the verification email is sent to"
const verificationTokenMissingText = "expected the plaintext token the verification link carries"
const frontendBaseURLMissingText = "expected the base URL the frontend is served from"
const frontendBaseURLUnparseableText = "failed parsing the base URL the frontend is served from"
const frontendBaseURLIncompleteText = "expected an absolute URL carrying both a scheme and a host"
const sendVerificationEmailFailedText = "failed sending the verification email"
const recipientIDMissingText = "expected the id of the user the message is sent to"
const saveMessageFailedText = "failed saving the message to the messaging log"
const verificationMessageTypeUnsupportedText = "expected a message type that carries a verification link"
const storedMessageErrorsUnreadableText = "failed decoding the stored message errors"

// ErrRecipientEmailMissing reports a verification email asked for without an address to send it to.
type ErrRecipientEmailMissing struct{}

func (e ErrRecipientEmailMissing) Error() string {
	return fmt.Sprintf("missing recipient email: %s", recipientEmailMissingText)
}

// ErrVerificationTokenMissing reports a verification email asked for without a challenge token.
type ErrVerificationTokenMissing struct{}

func (e ErrVerificationTokenMissing) Error() string {
	return fmt.Sprintf("missing verification token: %s", verificationTokenMissingText)
}

// ErrFrontendBaseURLMissing reports a verification email asked for without a frontend base URL.
type ErrFrontendBaseURLMissing struct{}

func (e ErrFrontendBaseURLMissing) Error() string {
	return fmt.Sprintf("missing frontend base url: %s", frontendBaseURLMissingText)
}

// ErrFrontendBaseURLUnparseable reports a frontend base URL that is not a URL at all.
type ErrFrontendBaseURLUnparseable struct {
	FrontendBaseURL string
	Cause           error
}

func (e ErrFrontendBaseURLUnparseable) Error() string {
	return fmt.Sprintf("invalid frontend base url %q: %s: %s",
		e.FrontendBaseURL, frontendBaseURLUnparseableText, e.Cause)
}

func (e ErrFrontendBaseURLUnparseable) Unwrap() error {
	return e.Cause
}

// ErrFrontendBaseURLIncomplete reports a frontend base URL parsed without a scheme or a host.
type ErrFrontendBaseURLIncomplete struct {
	FrontendBaseURL string
}

func (e ErrFrontendBaseURLIncomplete) Error() string {
	return fmt.Sprintf("invalid frontend base url %q: %s",
		e.FrontendBaseURL, frontendBaseURLIncompleteText)
}

// ErrSendVerificationEmailFailed reports a message the email gateway refused to send.
type ErrSendVerificationEmailFailed struct {
	RecipientEmail string
	Cause          error
}

func (e ErrSendVerificationEmailFailed) Error() string {
	return fmt.Sprintf("%s to %q: %s", sendVerificationEmailFailedText, e.RecipientEmail, e.Cause)
}

func (e ErrSendVerificationEmailFailed) Unwrap() error {
	return e.Cause
}

// ErrRecipientIDMissing reports a message asked for without the user it is sent to.
type ErrRecipientIDMissing struct{}

func (e ErrRecipientIDMissing) Error() string {
	return fmt.Sprintf("missing recipient id: %s", recipientIDMissingText)
}

// ErrSaveMessageFailed reports a sent message the messaging log failed to store.
type ErrSaveMessageFailed struct {
	MessageType string
	Cause       error
}

func (e ErrSaveMessageFailed) Error() string {
	return fmt.Sprintf("%s %q: %s", saveMessageFailedText, e.MessageType, e.Cause)
}

func (e ErrSaveMessageFailed) Unwrap() error {
	return e.Cause
}

// ErrStoredMessageErrorsUnreadable reports a messaging_log row whose errors column is not a string array.
type ErrStoredMessageErrorsUnreadable struct {
	MessageID string
	Cause     error
}

func (e ErrStoredMessageErrorsUnreadable) Error() string {
	return fmt.Sprintf("%s for message %s: %s", storedMessageErrorsUnreadableText, e.MessageID, e.Cause)
}

func (e ErrStoredMessageErrorsUnreadable) Unwrap() error {
	return e.Cause
}

// ErrVerificationMessageTypeUnsupported reports a verification email asked for with no link path.
type ErrVerificationMessageTypeUnsupported struct {
	Type string
}

func (e ErrVerificationMessageTypeUnsupported) Error() string {
	return fmt.Sprintf("message type %q: %s", e.Type, verificationMessageTypeUnsupportedText)
}
