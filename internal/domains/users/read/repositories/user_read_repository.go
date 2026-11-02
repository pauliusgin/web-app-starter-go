// Package repositories declares the user lookup port the read side depends on.
package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/models"
)

// UserRepository is the user lookup port the domain depends on.
type UserRepository interface {
	// GetUserReadModelByID loads the client-facing view of the user.
	GetUserReadModelByID(ctx context.Context, id uuid.UUID) (models.UserReadModel, error)
}
