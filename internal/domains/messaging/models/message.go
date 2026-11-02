// Package models defines the messages the application sends to users, as recorded in the messaging log.
package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// MessageType is what a message is about, e.g. verifying an email address.
type MessageType string

const (
	// EmailVerification carries the link that verifies a user's email address.
	EmailVerification MessageType = "email_verification"
	// PasswordReset carries the link that lets a user set a new password without the current one.
	PasswordReset MessageType = "password_reset"
)

// Method is the channel a message is delivered through.
type Method string

const (
	// MethodEmail delivers the message by email.
	MethodEmail Method = "email"
	// MethodSMS delivers the message as a text message.
	MethodSMS Method = "sms"
	// MethodPush delivers the message as a mobile push notification.
	MethodPush Method = "push"
	// MethodInApp delivers the message inside the application.
	MethodInApp Method = "in_app"
)

// Status is the outcome of handing a message to its delivery provider.
type Status string

const (
	// StatusSent marks a message the provider accepted.
	StatusSent Status = "sent"
	// StatusFailed marks a message the provider rejected or could not be reached for.
	StatusFailed Status = "failed"
)

// Message is a message as stored in the messaging log.
type Message struct {
	ID             uuid.UUID
	RecipientID    uuid.UUID
	RecipientEmail string
	Data           json.RawMessage
	Type           MessageType
	Method         Method
	Status         Status
	ReferenceID    string
	Errors         []string
	CreatedAt      time.Time
}

// NewMessage is a message about to be logged, before the database stamps CreatedAt.
type NewMessage struct {
	ID             uuid.UUID
	RecipientID    uuid.UUID
	RecipientEmail string
	Data           json.RawMessage
	Type           MessageType
	Method         Method
	Status         Status
	ReferenceID    string
	Errors         []string
}

// NewMessageParams is what a sender supplies to log a message; the id is generated, not passed.
type NewMessageParams struct {
	RecipientID    uuid.UUID
	RecipientEmail string
	Data           json.RawMessage
	Type           MessageType
	Method         Method
	Status         Status
	ReferenceID    string
	Errors         []string
}

// New builds a message to log, normalising missing data and errors to empty JSON values.
func New(params NewMessageParams) NewMessage {
	data := params.Data
	if data == nil {
		data = json.RawMessage(`{}`)
	}

	errors := params.Errors
	if errors == nil {
		errors = []string{}
	}

	return NewMessage{
		ID:             uuid.New(),
		RecipientID:    params.RecipientID,
		RecipientEmail: params.RecipientEmail,
		Data:           data,
		Type:           params.Type,
		Method:         params.Method,
		Status:         params.Status,
		ReferenceID:    params.ReferenceID,
		Errors:         errors,
	}
}
