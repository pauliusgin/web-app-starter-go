package routes

import (
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/tokens"
)

// TokenRoutePath is the base path every authentication route is declared under.
const TokenRoutePath = "/api/v1/tokens"

// TokenRoutesDependencies carries the collaborators the auth routes hand to their handlers.
type TokenRoutesDependencies struct {
	RotateRefreshToken tokens.RotateRefreshTokenDependencies
	RevokeRefreshToken tokens.RevokeRefreshTokenDependencies
}

// RegisterTokenRoutes adds the refresh token routes under TokenRoutePath to mux.
func RegisterTokenRoutes(mux *http.ServeMux, dependencies TokenRoutesDependencies) {
	mux.HandleFunc("POST "+TokenRoutePath+"/refresh",
		tokens.RotateRefreshToken(dependencies.RotateRefreshToken))

	mux.HandleFunc("POST "+TokenRoutePath+"/revoke",
		tokens.RevokeRefreshToken(dependencies.RevokeRefreshToken))
}
