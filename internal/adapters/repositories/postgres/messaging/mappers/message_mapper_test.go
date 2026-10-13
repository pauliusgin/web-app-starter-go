package mappers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/models"
)

var fixedCreatedAt = time.Date(2026, time.October, 6, 9, 0, 0, 0, time.UTC)

func messagingLogRow() sqlc.MessagingLog {
	return sqlc.MessagingLog{
		ID:             uuid.MustParse("6a1f3c2e-8b4d-4e7a-9c15-2d7e8f9a0b31"),
		RecipientID:    uuid.MustParse("1c9d4e2b-5f80-4a11-8b3c-2e7a9d6f4c15"),
		RecipientEmail: sql.NullString{String: "accountant@example.com", Valid: true},
		Data:           json.RawMessage(`{"challengeId":"0b3e2c1d-7f6a-4e5b-8c9d-1a2b3c4d5e6f"}`),
		Type:           "email_verification",
		Method:         "email",
		Status:         "failed",
		ReferenceID:    sql.NullString{},
		Errors:         json.RawMessage(`["brevo refused it","rate limited"]`),
		CreatedAt:      fixedCreatedAt,
	}
}

func TestMessageMapperMapsEveryColumn(t *testing.T) {
	row := messagingLogRow()

	mapped, err := MessageMapper(row)
	if err != nil {
		t.Fatalf("MessageMapper: %v", err)
	}

	want := models.Message{
		ID:             row.ID,
		RecipientID:    row.RecipientID,
		RecipientEmail: "accountant@example.com",
		Data:           row.Data,
		Type:           models.EmailVerification,
		Method:         models.MethodEmail,
		Status:         models.StatusFailed,
		ReferenceID:    "",
		Errors:         []string{"brevo refused it", "rate limited"},
		CreatedAt:      fixedCreatedAt,
	}
	if !reflect.DeepEqual(mapped, want) {
		t.Errorf("mapped = %#v, want %#v", mapped, want)
	}
}

func TestMessageMapperReportsErrorsThatAreNotAStringArray(t *testing.T) {
	row := messagingLogRow()
	row.Errors = json.RawMessage(`{}`)

	_, err := MessageMapper(row)

	unreadable := e.ErrStoredMessageErrorsUnreadable{}
	if !errors.As(err, &unreadable) {
		t.Fatalf("error = %v (%T), want %T", err, err, unreadable)
	}
	if unreadable.MessageID != row.ID.String() {
		t.Errorf("MessageID = %q, want %q", unreadable.MessageID, row.ID)
	}
}

func TestSaveMessageParamsMapperStoresEmptyOptionalStringsAsNull(t *testing.T) {
	newMessage := models.New(models.NewMessageParams{
		RecipientID: uuid.New(),
		Type:        models.EmailVerification,
		Method:      models.MethodEmail,
		Status:      models.StatusFailed,
		Errors:      []string{"brevo refused it"},
	})

	params := SaveMessageParamsMapper(newMessage)

	if params.RecipientEmail.Valid {
		t.Errorf("RecipientEmail = %#v, want NULL for an empty address", params.RecipientEmail)
	}
	if params.ReferenceID.Valid {
		t.Errorf("ReferenceID = %#v, want NULL when the provider gave none", params.ReferenceID)
	}
	if string(params.Errors) != `["brevo refused it"]` {
		t.Errorf("Errors = %s, want a JSON string array", params.Errors)
	}
	if string(params.Data) != `{}` {
		t.Errorf("Data = %s, want an empty JSON object", params.Data)
	}
}

func TestSaveMessageParamsMapperCarriesEveryField(t *testing.T) {
	newMessage := models.New(models.NewMessageParams{
		RecipientID:    uuid.New(),
		RecipientEmail: "accountant@example.com",
		Data:           json.RawMessage(`{"challengeId":"abc"}`),
		Type:           models.EmailVerification,
		Method:         models.MethodEmail,
		Status:         models.StatusSent,
		ReferenceID:    "brevo-42",
	})

	params := SaveMessageParamsMapper(newMessage)

	want := sqlc.SaveMessageParams{
		ID:             newMessage.ID,
		RecipientID:    newMessage.RecipientID,
		RecipientEmail: sql.NullString{String: "accountant@example.com", Valid: true},
		Data:           json.RawMessage(`{"challengeId":"abc"}`),
		Type:           "email_verification",
		Method:         "email",
		Status:         "sent",
		ReferenceID:    sql.NullString{String: "brevo-42", Valid: true},
		Errors:         json.RawMessage(`[]`),
	}
	if !reflect.DeepEqual(params, want) {
		t.Errorf("params = %#v, want %#v", params, want)
	}
}
