package email

import (
	"context"
	"fmt"
)

var _ Gateway = (*InMemoryGateway)(nil)

const inMemoryProviderMessageIDPrefix = "in-memory-"

// InMemoryGateway is a Gateway that records what it was asked to send instead of sending it.
type InMemoryGateway struct {
	Sent               []VerificationEmailPayload
	SentPasswordResets []PasswordResetEmailPayload

	// SendFailure, when set, is returned by every send instead of recording the message.
	SendFailure error
}

// NewInMemoryGateway builds a gateway with nothing sent yet and no failure staged.
func NewInMemoryGateway() *InMemoryGateway {
	return &InMemoryGateway{}
}

// SendVerificationEmail records the message and answers with a provider id numbered by send order
// ("in-memory-1", "in-memory-2", ...), or returns SendFailure when one is staged.
func (g *InMemoryGateway) SendVerificationEmail(
	_ context.Context,
	message VerificationEmailPayload,
) (Response, error) {
	if g.SendFailure != nil {
		return Response{}, g.SendFailure
	}

	g.Sent = append(g.Sent, message)

	return Response{ProviderMessageID: InMemoryProviderMessageID(g.sendCount())}, nil
}

// SendPasswordResetEmail records the message and answers with a provider id numbered by send order across
// every kind of email, or returns SendFailure when one is staged.
func (g *InMemoryGateway) SendPasswordResetEmail(
	_ context.Context,
	message PasswordResetEmailPayload,
) (Response, error) {
	if g.SendFailure != nil {
		return Response{}, g.SendFailure
	}

	g.SentPasswordResets = append(g.SentPasswordResets, message)

	return Response{ProviderMessageID: InMemoryProviderMessageID(g.sendCount())}, nil
}

func (g *InMemoryGateway) sendCount() int {
	return len(g.Sent) + len(g.SentPasswordResets)
}

// InMemoryProviderMessageID is the provider id the in-memory gateway answers its nth send with.
func InMemoryProviderMessageID(sendNumber int) string {
	return fmt.Sprintf("%s%d", inMemoryProviderMessageIDPrefix, sendNumber)
}

// OnlySentMessage returns the sent verification message, and false unless exactly one was sent.
func (g *InMemoryGateway) OnlySentMessage() (VerificationEmailPayload, bool) {
	if len(g.Sent) != 1 {
		return VerificationEmailPayload{}, false
	}

	return g.Sent[0], true
}

// OnlySentPasswordResetEmail returns the sent password reset message, and false unless exactly one was sent.
func (g *InMemoryGateway) OnlySentPasswordResetEmail() (PasswordResetEmailPayload, bool) {
	if len(g.SentPasswordResets) != 1 {
		return PasswordResetEmailPayload{}, false
	}

	return g.SentPasswordResets[0], true
}
