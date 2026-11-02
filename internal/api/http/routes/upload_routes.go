package routes

import (
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/uploads"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
)

// UploadRoutePath is the base path every upload route is declared under.
const UploadRoutePath = "/api/v1/uploads"

// UploadRoutesDependencies carries the collaborators the upload routes hand to their handlers.
type UploadRoutesDependencies struct {
	UploadFiles              uploads.UploadFilesDependencies
	RequireAuthenticatedUser middleware.RequireAuthenticatedUserDependencies
}

// RegisterUploadRoutes adds the upload routes under UploadRoutePath to mux.
func RegisterUploadRoutes(mux *http.ServeMux, dependencies UploadRoutesDependencies) {
	requireAuth := middleware.RequireAuthenticatedUser(dependencies.RequireAuthenticatedUser)

	mux.Handle("POST "+UploadRoutePath, requireAuth(uploads.UploadFiles(dependencies.UploadFiles)))
}
