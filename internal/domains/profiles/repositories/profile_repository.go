// Package repositories declares the profiles table port the profile use cases depend on.
package repositories

import (
	"context"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/models"
)

// ProfileRepository is the profiles table port the profile use cases record and change profiles through.
type ProfileRepository interface {
	// SaveProfile records the profile, or returns errors.ErrProfileAlreadyExists when the user has one.
	SaveProfile(ctx context.Context, newProfile models.NewProfile) (models.Profile, error)

	// UpdateProfile replaces the names, or returns errors.ErrProfileNotFound when no live profile exists.
	UpdateProfile(ctx context.Context, change models.UpdateProfileParams) (models.Profile, error)
}
