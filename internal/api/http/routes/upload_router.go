package routes

import (
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/uploads"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
)

// UploadRoutePath is the single path the upload routes own, and where BuildRouter mounts them.
const UploadRoutePath = "/api/v1/uploads"

// UploadRouterDependencies carries the collaborators the upload routes hand to their handlers.
type UploadRouterDependencies struct {
	UploadFiles              uploads.UploadFilesDependencies
	RequireAuthenticatedUser middleware.RequireAuthenticatedUserDependencies
}

// UploadRouter returns the routes under UploadRoutePath.
func UploadRouter(dependencies UploadRouterDependencies) http.Handler {
	mux := &http.ServeMux{}

	requireAuth := middleware.RequireAuthenticatedUser(dependencies.RequireAuthenticatedUser)

	mux.Handle("POST "+UploadRoutePath, requireAuth(uploads.UploadFiles(dependencies.UploadFiles)))

	return mux
}
