// Package repositories declares the profile lookup port the read side depends on.
package repositories

import (
	"context"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/read/models"
)

// ProfileReadRepository is the profile lookup port the profile queries answer clients through.
type ProfileReadRepository interface {
	// GetProfileReadModelByID returns the client-facing view of the user's live profile,
	// or errors.ErrProfileNotFound.
	GetProfileReadModelByID(ctx context.Context, userID uuid.UUID) (models.ProfileReadModel, error)
}
