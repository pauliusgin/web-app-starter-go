package routes

import (
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/messages"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
)

// MessageRoutePath is the base path every message route is declared under.
const MessageRoutePath = "/api/v1/messages"

// MessageRoutesDependencies carries the collaborators the message routes hand to their handlers.
type MessageRoutesDependencies struct {
	ResendEmailVerification  messages.ResendEmailVerificationDependencies
	SendPasswordResetEmail   messages.SendPasswordResetEmailDependencies
	RequireAuthenticatedUser middleware.RequireAuthenticatedUserDependencies
}

// RegisterMessageRoutes adds the message routes under MessageRoutePath to mux.
func RegisterMessageRoutes(mux *http.ServeMux, dependencies MessageRoutesDependencies) {
	requireAuth := middleware.RequireAuthenticatedUser(dependencies.RequireAuthenticatedUser)

	mux.Handle("POST "+MessageRoutePath+"/verification-email",
		requireAuth(messages.ResendEmailVerification(dependencies.ResendEmailVerification)))

	mux.HandleFunc("POST "+MessageRoutePath+"/password-reset-email",
		messages.SendPasswordResetEmail(dependencies.SendPasswordResetEmail))
}
