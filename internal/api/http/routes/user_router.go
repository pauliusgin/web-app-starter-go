package routes

import (
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/users"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
)

// UserRoutePrefix is the subtree the user routes own, and where BuildRouter mounts them.
const UserRoutePrefix = "/api/v1/users/"

// UserRouterDependencies carries the collaborators the user routes hand to their handlers.
type UserRouterDependencies struct {
	SignUp                   users.SignUpUserDependencies
	SignIn                   users.SignInUserDependencies
	VerifyEmail              users.VerifyEmailDependencies
	DeleteUser               users.DeleteUserDependencies
	RequestEmailChange       users.RequestEmailChangeDependencies
	ChangePassword           users.ChangePasswordDependencies
	RequireAuthenticatedUser middleware.RequireAuthenticatedUserDependencies
}

// UserRouter returns the routes under UserRoutePrefix.
func UserRouter(dependencies UserRouterDependencies) http.Handler {
	mux := &http.ServeMux{}

	requireAuth := middleware.RequireAuthenticatedUser(dependencies.RequireAuthenticatedUser)

	mux.HandleFunc("POST "+UserRoutePrefix+"signup", users.SignUpUser(dependencies.SignUp))

	mux.HandleFunc("POST "+UserRoutePrefix+"signin", users.SignInUser(dependencies.SignIn))

	mux.Handle("POST "+UserRoutePrefix+"verify-email", requireAuth(users.VerifyEmail(dependencies.VerifyEmail)))

	mux.Handle("POST "+UserRoutePrefix+"change-email",
		requireAuth(users.RequestEmailChange(dependencies.RequestEmailChange)))

	mux.Handle("POST "+UserRoutePrefix+"change-password",
		requireAuth(users.ChangePassword(dependencies.ChangePassword)))

	mux.Handle("DELETE "+UserRoutePrefix+"{"+users.UserIDPathValue+"}",
		requireAuth(users.DeleteUser(dependencies.DeleteUser)))

	return mux
}
