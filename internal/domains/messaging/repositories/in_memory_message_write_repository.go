package repositories

import (
	"context"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/models"
)

var _ MessageWriteRepository = (*InMemoryMessageWriteRepository)(nil)

// InMemoryMessageWriteRepository is a MessageWriteRepository that keeps saved messages in a slice.
type InMemoryMessageWriteRepository struct {
	Saved []models.NewMessage

	// SaveFailure, when set, is returned instead of saving the message.
	SaveFailure error
}

// NewInMemoryMessageWriteRepository builds a repository with nothing saved and no failure staged.
func NewInMemoryMessageWriteRepository() *InMemoryMessageWriteRepository {
	return &InMemoryMessageWriteRepository{}
}

// Save records the message, or returns SaveFailure when one is staged.
func (r *InMemoryMessageWriteRepository) Save(
	_ context.Context,
	newMessage models.NewMessage,
) (models.Message, error) {
	if r.SaveFailure != nil {
		return models.Message{}, r.SaveFailure
	}

	r.Saved = append(r.Saved, newMessage)

	return models.Message{
		ID:             newMessage.ID,
		RecipientID:    newMessage.RecipientID,
		RecipientEmail: newMessage.RecipientEmail,
		Data:           newMessage.Data,
		Type:           newMessage.Type,
		Method:         newMessage.Method,
		Status:         newMessage.Status,
		ReferenceID:    newMessage.ReferenceID,
		Errors:         newMessage.Errors,
	}, nil
}

// OnlySavedMessage returns the saved message, and false unless exactly one was saved.
func (r *InMemoryMessageWriteRepository) OnlySavedMessage() (models.NewMessage, bool) {
	if len(r.Saved) != 1 {
		return models.NewMessage{}, false
	}

	return r.Saved[0], true
}
