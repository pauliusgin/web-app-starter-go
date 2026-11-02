package routes

import (
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/profiles"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
)

// ProfileRoutePath is the base path every profile route is declared under.
const ProfileRoutePath = "/api/v1/profiles"

// ProfileRoutesDependencies carries the collaborators the profile routes hand to their handlers.
type ProfileRoutesDependencies struct {
	CreateProfile            profiles.CreateProfileDependencies
	UpdateProfile            profiles.UpdateProfileDependencies
	RequireAuthenticatedUser middleware.RequireAuthenticatedUserDependencies
}

// RegisterProfileRoutes adds the profile routes under ProfileRoutePath to mux.
func RegisterProfileRoutes(mux *http.ServeMux, dependencies ProfileRoutesDependencies) {
	requireAuth := middleware.RequireAuthenticatedUser(dependencies.RequireAuthenticatedUser)

	mux.Handle("POST "+ProfileRoutePath, requireAuth(profiles.CreateProfile(dependencies.CreateProfile)))

	mux.Handle("PUT "+ProfileRoutePath, requireAuth(profiles.UpdateProfile(dependencies.UpdateProfile)))
}
