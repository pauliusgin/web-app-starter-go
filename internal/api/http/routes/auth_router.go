package routes

import (
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/tokens"
)

// AuthRoutePrefix is the subtree the authentication routes own, and where BuildRouter mounts them.
const AuthRoutePrefix = "/api/v1/tokens/"

// AuthRouterDependencies carries the collaborators the auth routes hand to their handlers.
type AuthRouterDependencies struct {
	RotateRefreshToken tokens.RotateDependencies
	RevokeRefreshToken tokens.RevokeRefreshTokenDependencies
}

// AuthRouter returns the routes under AuthRoutePrefix.
func AuthRouter(dependencies AuthRouterDependencies) http.Handler {
	mux := &http.ServeMux{}

	mux.HandleFunc("POST "+AuthRoutePrefix+"refresh",
		tokens.RotateRefreshToken(dependencies.RotateRefreshToken))
	mux.HandleFunc("POST "+AuthRoutePrefix+"revoke",
		tokens.RevokeRefreshToken(dependencies.RevokeRefreshToken))

	return mux
}
