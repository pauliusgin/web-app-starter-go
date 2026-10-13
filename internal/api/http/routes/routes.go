// Package routes declares the HTTP routing table.
package routes

import (
	"net/http"
)

// RouterDependencies carries the collaborators each sub-router hands to its handlers.
type RouterDependencies struct {
	Users    UserRouterDependencies
	Auth     AuthRouterDependencies
	Uploads  UploadRouterDependencies
	Profiles ProfileRouterDependencies
}

// BuildRouter returns the router with every sub-router of the API mounted under its own prefix.
func BuildRouter(dependencies RouterDependencies) http.Handler {
	mux := &http.ServeMux{}

	mux.Handle(HealthPath, HealthRouter())
	mux.Handle(UserRoutePrefix, UserRouter(dependencies.Users))
	mux.Handle(AuthRoutePrefix, AuthRouter(dependencies.Auth))
	mux.Handle(UploadRoutePath, UploadRouter(dependencies.Uploads))
	mux.Handle(ProfileRoutePath, ProfileRouter(dependencies.Profiles))

	return mux
}
