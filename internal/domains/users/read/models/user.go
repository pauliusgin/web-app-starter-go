// Package models holds the user shapes the read side returns.
package models

import (
	"time"

	"github.com/google/uuid"
)

// UserReadModel is a returned user as clients see it; nil fields are not recorded yet.
type UserReadModel struct {
	ID              uuid.UUID
	Email           string
	CreatedAt       time.Time
	EmailVerifiedAt *time.Time
	SigninAt        *time.Time
	FirstName       *string
	LastName        *string
}
