// Package routes declares the HTTP routing table.
package routes

import (
	"net/http"
)

// RouterDependencies carries the collaborators each group of routes hands to its handlers.
type RouterDependencies struct {
	Users    UserRoutesDependencies
	Auth     TokenRoutesDependencies
	Uploads  UploadRoutesDependencies
	Profiles ProfileRoutesDependencies
}

// BuildRouter returns one mux holding every API route under its full pattern.
func BuildRouter(dependencies RouterDependencies) http.Handler {
	mux := &http.ServeMux{}

	RegisterHealthRoutes(mux)
	RegisterUserRoutes(mux, dependencies.Users)
	RegisterTokenRoutes(mux, dependencies.Auth)
	RegisterUploadRoutes(mux, dependencies.Uploads)
	RegisterProfileRoutes(mux, dependencies.Profiles)

	return mux
}
