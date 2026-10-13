package routes

import (
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/health"
)

// HealthPath is the single path the health router answers, and where BuildRouter mounts it.
const HealthPath = "/api/health"

// HealthRouter returns the liveness route.
func HealthRouter() http.Handler {
	mux := &http.ServeMux{}

	mux.HandleFunc("GET "+HealthPath, health.Report())

	return mux
}
