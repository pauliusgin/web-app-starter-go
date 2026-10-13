package routes

import (
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/profiles"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
)

// ProfileRoutePath is the single path the profile routes own, and where BuildRouter mounts them.
const ProfileRoutePath = "/api/v1/profiles"

// ProfileRouterDependencies carries the collaborators the profile routes hand to their handlers.
type ProfileRouterDependencies struct {
	CreateProfile            profiles.CreateProfileDependencies
	UpdateProfile            profiles.UpdateProfileDependencies
	RequireAuthenticatedUser middleware.RequireAuthenticatedUserDependencies
}

// ProfileRouter returns the routes under ProfileRoutePath.
func ProfileRouter(dependencies ProfileRouterDependencies) http.Handler {
	mux := &http.ServeMux{}

	requireAuth := middleware.RequireAuthenticatedUser(dependencies.RequireAuthenticatedUser)

	mux.Handle("POST "+ProfileRoutePath, requireAuth(profiles.CreateProfile(dependencies.CreateProfile)))

	mux.Handle("PUT "+ProfileRoutePath, requireAuth(profiles.UpdateProfile(dependencies.UpdateProfile)))

	return mux
}
