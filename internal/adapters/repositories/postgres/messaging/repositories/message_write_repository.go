// Package repositories adapts the generated sqlc queries to the messaging log ports.
package repositories

import (
	"context"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/messaging/mappers"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/models"
)

var _ MessageWriteRepository = MessageWriteRepository{}

// MessageWriteRepository stores sent messages through the generated sqlc queries.
type MessageWriteRepository struct {
	queries *sqlc.Queries
}

// NewMessageWriteRepository builds a repository that writes messages through the given sqlc queries.
func NewMessageWriteRepository(queries *sqlc.Queries) MessageWriteRepository {
	return MessageWriteRepository{queries: queries}
}

// Save inserts the message and returns the row the database produced.
func (r MessageWriteRepository) Save(ctx context.Context, newMessage models.NewMessage) (models.Message, error) {
	row, err := r.queries.SaveMessage(ctx, mappers.SaveMessageParamsMapper(newMessage))
	if err != nil {
		return models.Message{}, e.ErrSaveMessageFailed{MessageType: string(newMessage.Type), Cause: err}
	}

	return mappers.MessageMapper(row)
}
