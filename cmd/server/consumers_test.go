package main

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/gateways/email"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/usecases"
)

func TestRegistryDependenciesWireVerificationEmailWhenGatewayPresent(t *testing.T) {
	silenceDefaultLogger(t)

	gateway := email.NewInMemoryGateway()
	messageRepository := repositories.NewInMemoryMessageWriteRepository()
	dependencies := buildRegistryDependencies(startEventConsumersParams{
		FrontendBaseURL:   "https://app.example.com",
		EmailGateway:      gateway,
		MessageRepository: messageRepository,
	})

	sendVerificationEmail := dependencies.VerificationEmail.SendVerificationEmail
	if sendVerificationEmail == nil {
		t.Fatal("SendVerificationEmail = nil, want a sender wired over the gateway")
	}

	err := sendVerificationEmail(context.Background(), usecases.SendVerificationEmailParams{
		RecipientID:    uuid.New(),
		ChallengeID:    uuid.New(),
		RecipientEmail: "someone@example.com",
		Token:          "abc123",
		Type:           models.EmailVerification,
	})
	if err != nil {
		t.Fatalf("SendVerificationEmail: %v", err)
	}

	message, ok := gateway.OnlySentMessage()
	if !ok {
		t.Fatalf("sent messages = %d, want 1", len(gateway.Sent))
	}
	if !strings.HasPrefix(message.VerificationLink, "https://app.example.com/") {
		t.Errorf("VerificationLink = %q, want it built from the configured frontend base URL",
			message.VerificationLink)
	}

	if _, ok := messageRepository.OnlySavedMessage(); !ok {
		t.Errorf("saved messages = %d, want the send logged through the wired repository",
			len(messageRepository.Saved))
	}
}
