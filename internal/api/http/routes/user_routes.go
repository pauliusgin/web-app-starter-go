package routes

import (
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/users"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
)

// UserRoutePath is the base path every user route is declared under.
const UserRoutePath = "/api/v1/users"

// UserRoutesDependencies carries the collaborators the user routes hand to their handlers.
type UserRoutesDependencies struct {
	SignUpUser               users.SignUpUserDependencies
	SignInUser               users.SignInUserDependencies
	SignUpUserGoogle         users.SignUpUserGoogleDependencies
	SignInUserGoogle         users.SignInUserGoogleDependencies
	GetGoogleSignInConfig    users.GetGoogleSignInConfigDependencies
	VerifyEmail              users.VerifyEmailDependencies
	GetUserByID              users.GetUserByIDDependencies
	DeleteUser               users.DeleteUserDependencies
	RequestEmailChange       users.RequestEmailChangeDependencies
	ChangePassword           users.ChangePasswordDependencies
	AddPassword              users.AddPasswordDependencies
	AddGoogleAccount         users.AddGoogleAccountDependencies
	ResetPassword            users.ResetPasswordDependencies
	RequireAuthenticatedUser middleware.RequireAuthenticatedUserDependencies
}

// RegisterUserRoutes adds the user routes under UserRoutePath to mux.
func RegisterUserRoutes(mux *http.ServeMux, dependencies UserRoutesDependencies) {
	requireAuth := middleware.RequireAuthenticatedUser(dependencies.RequireAuthenticatedUser)

	mux.HandleFunc("POST "+UserRoutePath+"/signup", users.SignUpUser(dependencies.SignUpUser))

	mux.HandleFunc("POST "+UserRoutePath+"/signin", users.SignInUser(dependencies.SignInUser))

	mux.HandleFunc("POST "+UserRoutePath+"/google-signup", users.SignUpUserGoogle(dependencies.SignUpUserGoogle))

	mux.HandleFunc("POST "+UserRoutePath+"/google-signin", users.SignInUserGoogle(dependencies.SignInUserGoogle))

	mux.HandleFunc("GET "+UserRoutePath+"/google-config",
		users.GetGoogleSignInConfig(dependencies.GetGoogleSignInConfig))

	mux.HandleFunc("POST "+UserRoutePath+"/reset-password", users.ResetPassword(dependencies.ResetPassword))

	mux.Handle("POST "+UserRoutePath+"/verify-email", requireAuth(users.VerifyEmail(dependencies.VerifyEmail)))

	mux.Handle("POST "+UserRoutePath+"/change-email",
		requireAuth(users.RequestEmailChange(dependencies.RequestEmailChange)))

	mux.Handle("POST "+UserRoutePath+"/change-password",
		requireAuth(users.ChangePassword(dependencies.ChangePassword)))

	mux.Handle("POST "+UserRoutePath+"/add-password",
		requireAuth(users.AddPassword(dependencies.AddPassword)))

	mux.Handle("POST "+UserRoutePath+"/add-google",
		requireAuth(users.AddGoogleAccount(dependencies.AddGoogleAccount)))

	mux.Handle("GET "+UserRoutePath+"/me",
		requireAuth(users.GetUserByID(dependencies.GetUserByID)))

	mux.Handle("DELETE "+UserRoutePath+"/me",
		requireAuth(users.DeleteUser(dependencies.DeleteUser)))
}
