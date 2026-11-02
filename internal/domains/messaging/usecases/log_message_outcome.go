package usecases

import (
	"context"
	"log/slog"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	messagingevents "github.com/pauliusgin/web_app_starter-be/internal/events/messaging"
)

const messageRepositoryMissingText = "no message repository wired, message not logged"
const saveMessageFailedText = "failed saving the message to the messaging log"

type logMessageOutcomeDependencies struct {
	MessageRepository repositories.MessageWriteRepository
	EventDispatcher   events.Dispatcher
	Logger            *slog.Logger
}

func logMessageOutcome(ctx context.Context, d logMessageOutcomeDependencies, newMessage models.NewMessage) {
	logger := d.Logger
	if logger == nil {
		logger = slog.Default()
	}

	saveMessage(ctx, saveMessageParams{
		Repository: d.MessageRepository,
		Logger:     logger,
		NewMessage: newMessage,
	})

	dispatchMessageOutcome(ctx, dispatchMessageOutcomeParams{EventDispatcher: d.EventDispatcher, NewMessage: newMessage})
}

type saveMessageParams struct {
	Repository repositories.MessageWriteRepository
	Logger     *slog.Logger
	NewMessage models.NewMessage
}

func saveMessage(ctx context.Context, params saveMessageParams) {
	if params.Repository == nil {
		params.Logger.Error(messageRepositoryMissingText,
			"messageId", params.NewMessage.ID, "messageType", params.NewMessage.Type)
		return
	}

	if _, err := params.Repository.SaveMessage(ctx, params.NewMessage); err != nil {
		params.Logger.Error(saveMessageFailedText,
			"messageId", params.NewMessage.ID,
			"recipientId", params.NewMessage.RecipientID,
			"messageType", params.NewMessage.Type,
			"status", params.NewMessage.Status,
			"error", err)
	}
}

type dispatchMessageOutcomeParams struct {
	EventDispatcher events.Dispatcher
	NewMessage      models.NewMessage
}

func dispatchMessageOutcome(ctx context.Context, params dispatchMessageOutcomeParams) {
	newMessage, dispatcher := params.NewMessage, params.EventDispatcher
	switch newMessage.Status {
	case models.StatusSent:
		messagingevents.DispatchMessageSent(ctx, dispatcher, messagingevents.NewMessageSentParams{
			MessageID:      newMessage.ID,
			RecipientID:    newMessage.RecipientID,
			RecipientEmail: newMessage.RecipientEmail,
			Type:           string(newMessage.Type),
			Method:         string(newMessage.Method),
			ReferenceID:    newMessage.ReferenceID,
		})
	case models.StatusFailed:
		messagingevents.DispatchMessageFailed(ctx, dispatcher, messagingevents.NewMessageFailedParams{
			MessageID:      newMessage.ID,
			RecipientID:    newMessage.RecipientID,
			RecipientEmail: newMessage.RecipientEmail,
			Type:           string(newMessage.Type),
			Method:         string(newMessage.Method),
			Errors:         newMessage.Errors,
		})
	}
}

func messageStatusFor(sendErr error) models.Status {
	if sendErr != nil {
		return models.StatusFailed
	}

	return models.StatusSent
}

func describeSendErrors(sendErr error) []string {
	if sendErr == nil {
		return []string{}
	}

	joined, isJoined := sendErr.(interface{ Unwrap() []error })
	if !isJoined {
		return []string{sendErr.Error()}
	}

	causes := joined.Unwrap()
	descriptions := make([]string, 0, len(causes))
	for _, cause := range causes {
		descriptions = append(descriptions, cause.Error())
	}

	return descriptions
}
