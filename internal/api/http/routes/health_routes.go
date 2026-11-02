package routes

import (
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/health"
)

// HealthRoutePath is the single path the liveness route answers.
const HealthRoutePath = "/api/health"

// RegisterHealthRoutes adds the liveness route at HealthRoutePath to mux.
func RegisterHealthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET "+HealthRoutePath, health.Report())
}
