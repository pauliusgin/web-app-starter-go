// Package mappers converts between messaging_log rows and domain messages.
package mappers

import (
	"encoding/json"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlnull"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/models"
)

// MessageMapper maps a stored messaging_log row onto the domain type.
func MessageMapper(row sqlc.MessagingLog) (models.Message, error) {
	storedErrors := []string{}
	if err := json.Unmarshal(row.Errors, &storedErrors); err != nil {
		return models.Message{}, e.ErrStoredMessageErrorsUnreadable{MessageID: row.ID.String(), Cause: err}
	}

	mapped := models.Message{
		ID:             row.ID,
		RecipientID:    row.RecipientID,
		RecipientEmail: row.RecipientEmail.String,
		Data:           row.Data,
		Type:           models.MessageType(row.Type),
		Method:         models.Method(row.Method),
		Status:         models.Status(row.Status),
		ReferenceID:    row.ReferenceID.String,
		Errors:         storedErrors,
		CreatedAt:      row.CreatedAt,
	}

	return mapped, nil
}

// SaveMessageParamsMapper maps a message about to be logged onto the insert query's parameters,
// storing empty optional strings as NULL.
func SaveMessageParamsMapper(newMessage models.NewMessage) sqlc.SaveMessageParams {
	// Marshalling a string slice cannot fail.
	encodedErrors, _ := json.Marshal(newMessage.Errors)

	return sqlc.SaveMessageParams{
		ID:             newMessage.ID,
		RecipientID:    newMessage.RecipientID,
		RecipientEmail: sqlnull.StringUnlessEmpty(newMessage.RecipientEmail),
		Data:           newMessage.Data,
		Type:           string(newMessage.Type),
		Method:         string(newMessage.Method),
		Status:         string(newMessage.Status),
		ReferenceID:    sqlnull.StringUnlessEmpty(newMessage.ReferenceID),
		Errors:         encodedErrors,
	}
}
