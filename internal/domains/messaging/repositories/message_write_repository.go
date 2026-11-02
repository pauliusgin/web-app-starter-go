// Package repositories declares the messaging log ports the messaging use cases depend on.
package repositories

import (
	"context"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/models"
)

// MessageWriteRepository is the messaging_log write port the messaging use cases record sends through.
type MessageWriteRepository interface {
	// SaveMessage records the message in the messaging log and returns it as stored.
	SaveMessage(ctx context.Context, newMessage models.NewMessage) (models.Message, error)
}
