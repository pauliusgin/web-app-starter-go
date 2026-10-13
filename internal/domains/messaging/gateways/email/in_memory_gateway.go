package email

import (
	"context"
	"fmt"
)

const inMemoryProviderMessageIDPrefix = "in-memory-"

// InMemoryGateway is a Gateway that records what it was asked to send instead of sending it.
type InMemoryGateway struct {
	Sent        []VerificationEmailPayload
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

	return Response{ProviderMessageID: InMemoryProviderMessageID(len(g.Sent))}, nil
}

// InMemoryProviderMessageID is the provider id the in-memory gateway answers its nth send with.
func InMemoryProviderMessageID(sendNumber int) string {
	return fmt.Sprintf("%s%d", inMemoryProviderMessageIDPrefix, sendNumber)
}

// OnlySentMessage returns the sent message, and false unless exactly one was sent.
func (g *InMemoryGateway) OnlySentMessage() (VerificationEmailPayload, bool) {
	if len(g.Sent) != 1 {
		return VerificationEmailPayload{}, false
	}

	return g.Sent[0], true
}
