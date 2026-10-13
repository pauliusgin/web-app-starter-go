package email

import (
	"context"
	"errors"
	"testing"
)

func testVerificationMessage() VerificationEmailPayload {
	return VerificationEmailPayload{
		RecipientEmail:   "accountant@example.com",
		VerificationLink: "https://app.example.com/verify-email?token=abc123",
	}
}

func TestInMemoryGatewayRecordsWhatItWasAskedToSend(t *testing.T) {
	gateway := NewInMemoryGateway()

	sentEmail, err := gateway.SendVerificationEmail(context.Background(), testVerificationMessage())
	if err != nil {
		t.Fatalf("SendVerificationEmail: %v", err)
	}
	if sentEmail.ProviderMessageID != "in-memory-1" {
		t.Errorf("ProviderMessageID = %q, want %q", sentEmail.ProviderMessageID, "in-memory-1")
	}

	message, ok := gateway.OnlySentMessage()
	if !ok {
		t.Fatalf("sent messages = %d, want 1", len(gateway.Sent))
	}
	if message != testVerificationMessage() {
		t.Errorf("sent message = %#v, want %#v", message, testVerificationMessage())
	}
}

func TestInMemoryGatewayReturnsTheStagedFailureAndRecordsNothing(t *testing.T) {
	sendFailure := errors.New("brevo refused it")
	gateway := NewInMemoryGateway()
	gateway.SendFailure = sendFailure

	sentEmail, err := gateway.SendVerificationEmail(context.Background(), testVerificationMessage())

	if !errors.Is(err, sendFailure) {
		t.Errorf("error = %v, want the staged failure", err)
	}
	if sentEmail != (Response{}) {
		t.Errorf("sent email = %#v, want the zero value for a refused send", sentEmail)
	}
	if len(gateway.Sent) != 0 {
		t.Errorf("sent messages = %d, want none recorded when the send failed", len(gateway.Sent))
	}
}

func TestOnlySentMessageReportsWhenThereIsNotExactlyOne(t *testing.T) {
	gateway := NewInMemoryGateway()

	if _, ok := gateway.OnlySentMessage(); ok {
		t.Error("OnlySentMessage reported a message, want none before anything was sent")
	}

	for range 2 {
		if _, err := gateway.SendVerificationEmail(context.Background(), testVerificationMessage()); err != nil {
			t.Fatalf("SendVerificationEmail: %v", err)
		}
	}

	if _, ok := gateway.OnlySentMessage(); ok {
		t.Error("OnlySentMessage reported one message, want it to refuse two")
	}
}
