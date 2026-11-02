package routes

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/profiles"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/tokens"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/uploads"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/users"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
	jwt "github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
	profilereadmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/read/models"
	profilequeries "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/read/queries"
	profilemodels "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/models"
	profileusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/write/usecases"
	uploadqueries "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/read/queries"
	uploadmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/models"
	uploadusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/write/usecases"
	readmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/models"
	userqueries "github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/queries"
	models "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

const signupPath = "/api/v1/users/signup"
const signinPath = "/api/v1/users/signin"
const refreshTokenPath = "/api/v1/tokens/refresh"
const revokeTokenPath = "/api/v1/tokens/revoke"

const verifyEmailPath = "/api/v1/users/verify-email?token=Kq7wL2mZ9xR4tV6yB1nH8pD3sG5jF0aC"

const requestEmailChangePath = "/api/v1/users/change-email"

const requestEmailChangeBody = `{"newEmail":"bookkeeper@example.com"}`

const changePasswordPath = "/api/v1/users/change-password"

const changePasswordBody = `{"currentPassword":"correct-horse","newPassword":"new-ledger-password"}`

const currentUserPath = "/api/v1/users/me"

const bearerCredential = "Bearer header.payload.signature"

const presentedRefreshToken = "Zx7cV2bN5mQ8wE1rT4yU7iO0pA3sD6fG9hJ2kL5zX8c"

const uploadFilesPath = "/api/v1/uploads"

const uploadBoundary = "routes-test-boundary"

const uploadContentType = "multipart/form-data; boundary=" + uploadBoundary

const uploadFilesBody = "--" + uploadBoundary + "\r\n" +
	"Content-Disposition: form-data; name=\"purpose\"\r\n\r\nprofile_picture\r\n" +
	"--" + uploadBoundary + "--\r\n"

const profilesPath = "/api/v1/profiles"

const profileBody = `{"firstName":"Ada","lastName":"Lovelace"}`

const signupBody = `{"email":"accountant@example.com","password":"correct-horse"}`

var routedUserID = uuid.MustParse("6f1c9f2e-4a1b-4c3d-9e8f-0a1b2c3d4e5f")

type routeRequest struct {
	method        string
	path          string
	body          string
	contentType   string
	authorization string
	refreshToken  string
}

func serveRouteRequest(router http.Handler, routed routeRequest) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(routed.method, routed.path, strings.NewReader(routed.body))
	contentType := routed.contentType
	if contentType == "" {
		contentType = "application/json"
	}
	request.Header.Set("Content-Type", contentType)
	if routed.authorization != "" {
		request.Header.Set("Authorization", routed.authorization)
	}
	if routed.refreshToken != "" {
		request.AddCookie(&http.Cookie{Name: tokens.RefreshTokenCookieName, Value: routed.refreshToken})
	}

	router.ServeHTTP(recorder, request)

	return recorder
}

func silenceRouteLogger(t *testing.T) {
	t.Helper()

	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
}

type fakeAccessTokens struct{}

func (fakeAccessTokens) IssueAccessToken(jwt.IssueAccessTokenParams) (string, error) {
	return "header.payload.signature", nil
}

func (fakeAccessTokens) ParseAccessToken(string) (jwt.AccessTokenClaims, error) {
	return jwt.AccessTokenClaims{UserID: routedUserID}, nil
}

func findRoutedUserByID(_ context.Context, id uuid.UUID) (models.User, error) {
	return models.User{ID: id, Email: "accountant@example.com"}, nil
}

func getRoutedUserReadModel(_ context.Context, id uuid.UUID) (readmodels.UserReadModel, error) {
	return readmodels.UserReadModel{ID: id, Email: "accountant@example.com"}, nil
}

func getRoutedProfile(
	_ context.Context,
	params profilequeries.GetProfileByIDParams,
) (profilequeries.GetProfileByIDResult, error) {
	return profilequeries.GetProfileByIDResult{Profile: profilereadmodels.ProfileReadModel{ID: params.UserID}}, nil
}

func routerWithEveryRouteWired() http.Handler {
	return BuildRouter(RouterDependencies{
		Users: UserRoutesDependencies{
			SignUpUser: users.SignUpUserDependencies{
				SignUpUser: func(_ context.Context, params usecases.SignUpUserParams) (usecases.SignUpUserResult, error) {
					return usecases.SignUpUserResult{User: models.User{ID: routedUserID, Email: params.Email}}, nil
				},
				GetUserReadModelByID: getRoutedUserReadModel,
				IssueRefreshToken: func(context.Context, refreshtoken.IssueParams) (refreshtoken.IssueResult, error) {
					return refreshtoken.IssueResult{Plaintext: "Qw9eR2tY5uI8oP1aS4dF7gH0jK3lZ6xC9vB2nM5qW8e"}, nil
				},
				Jwt: fakeAccessTokens{},
			},
			SignInUser: users.SignInUserDependencies{
				SignInUser: func(_ context.Context, params usecases.SignInUserParams) (usecases.SignInUserResult, error) {
					return usecases.SignInUserResult{User: models.User{ID: routedUserID, Email: params.Email}}, nil
				},
				GetUserReadModelByID: getRoutedUserReadModel,
				IssueRefreshToken: func(context.Context, refreshtoken.IssueParams) (refreshtoken.IssueResult, error) {
					return refreshtoken.IssueResult{Plaintext: "Qw9eR2tY5uI8oP1aS4dF7gH0jK3lZ6xC9vB2nM5qW8e"}, nil
				},
				Jwt: fakeAccessTokens{},
			},
			VerifyEmail: users.VerifyEmailDependencies{
				VerifyEmail: func(context.Context, usecases.VerifyEmailParams) error { return nil },
			},
			GetUserByID: users.GetUserByIDDependencies{
				GetUserByID: func(_ context.Context, params userqueries.GetUserByIDParams) (userqueries.GetUserByIDResult, error) {
					user, err := getRoutedUserReadModel(context.Background(), params.UserID)
					return userqueries.GetUserByIDResult{User: user}, err
				},
			},
			DeleteUser: users.DeleteUserDependencies{
				DeleteUser: func(context.Context, usecases.DeleteUserParams) error { return nil },
			},
			RequestEmailChange: users.RequestEmailChangeDependencies{
				RequestEmailChange: func(context.Context, usecases.RequestEmailChangeParams) error { return nil },
			},
			ChangePassword: users.ChangePasswordDependencies{
				ChangePassword: func(context.Context, usecases.ChangePasswordParams) error { return nil },
			},
			RequireAuthenticatedUser: middleware.RequireAuthenticatedUserDependencies{
				ParseAccessToken: fakeAccessTokens{}.ParseAccessToken,
				FindUserByID:     findRoutedUserByID,
			},
		},
		Auth: TokenRoutesDependencies{
			RotateRefreshToken: tokens.RotateRefreshTokenDependencies{
				RotateRefreshToken: func(context.Context, refreshtoken.RotateParams) (refreshtoken.RotateResult, error) {
					return refreshtoken.RotateResult{UserID: routedUserID, Plaintext: "rotated-refresh-token"}, nil
				},
				FindUserByID: findRoutedUserByID,
				Jwt:          fakeAccessTokens{},
			},
			RevokeRefreshToken: tokens.RevokeRefreshTokenDependencies{
				RevokeRefreshToken: func(context.Context, refreshtoken.RevokeParams) error { return nil },
			},
		},
		Uploads: UploadRoutesDependencies{
			UploadFiles: uploads.UploadFilesDependencies{
				UploadFiles: func(context.Context, uploadusecases.UploadFilesParams) (uploadusecases.UploadFilesResult, error) {
					return uploadusecases.UploadFilesResult{Uploads: []uploadmodels.Upload{{ID: routedUserID}}}, nil
				},
				GetUploadsByIDs: func(context.Context, uploadqueries.GetUploadsByIDsParams) (uploadqueries.GetUploadsByIDsResult, error) {
					return uploadqueries.GetUploadsByIDsResult{}, nil
				},
			},
			RequireAuthenticatedUser: middleware.RequireAuthenticatedUserDependencies{
				ParseAccessToken: fakeAccessTokens{}.ParseAccessToken,
				FindUserByID:     findRoutedUserByID,
			},
		},
		Profiles: ProfileRoutesDependencies{
			CreateProfile: profiles.CreateProfileDependencies{
				CreateProfile: func(context.Context, profileusecases.CreateProfileParams) (profileusecases.CreateProfileResult, error) {
					return profileusecases.CreateProfileResult{Profile: profilemodels.Profile{ID: routedUserID}}, nil
				},
				GetProfileByID: getRoutedProfile,
			},
			UpdateProfile: profiles.UpdateProfileDependencies{
				UpdateProfile: func(context.Context, profileusecases.UpdateProfileParams) (profileusecases.UpdateProfileResult, error) {
					return profileusecases.UpdateProfileResult{Profile: profilemodels.Profile{ID: routedUserID}}, nil
				},
				GetProfileByID: getRoutedProfile,
			},
			RequireAuthenticatedUser: middleware.RequireAuthenticatedUserDependencies{
				ParseAccessToken: fakeAccessTokens{}.ParseAccessToken,
				FindUserByID:     findRoutedUserByID,
			},
		},
	})
}

func TestRefreshTokenCookiePathMatchesTheTokenRoutes(t *testing.T) {
	if tokens.RefreshTokenCookiePath != TokenRoutePath {
		t.Errorf("tokens.RefreshTokenCookiePath = %q, want TokenRoutePath %q so the browser sends the cookie to the token routes",
			tokens.RefreshTokenCookiePath, TokenRoutePath)
	}
}

func TestBuildRouterReachesEveryRoute(t *testing.T) {
	cases := []struct {
		name       string
		request    routeRequest
		wantStatus int
	}{
		{name: "health", request: routeRequest{method: http.MethodGet, path: HealthRoutePath},
			wantStatus: http.StatusOK},
		{name: "signup", request: routeRequest{method: http.MethodPost, path: signupPath, body: signupBody},
			wantStatus: http.StatusCreated},
		{name: "signin", request: routeRequest{method: http.MethodPost, path: signinPath, body: signupBody},
			wantStatus: http.StatusOK},
		{name: "verify email", request: routeRequest{method: http.MethodPost, path: verifyEmailPath,
			authorization: bearerCredential}, wantStatus: http.StatusOK},
		{name: "get current user", request: routeRequest{method: http.MethodGet, path: currentUserPath,
			authorization: bearerCredential}, wantStatus: http.StatusOK},
		{name: "delete user", request: routeRequest{method: http.MethodDelete, path: currentUserPath,
			authorization: bearerCredential}, wantStatus: http.StatusOK},
		{name: "request email change", request: routeRequest{method: http.MethodPost, path: requestEmailChangePath,
			body: requestEmailChangeBody, authorization: bearerCredential}, wantStatus: http.StatusAccepted},
		{name: "change password", request: routeRequest{method: http.MethodPost, path: changePasswordPath,
			body: changePasswordBody, authorization: bearerCredential}, wantStatus: http.StatusOK},
		{name: "token refresh", request: routeRequest{method: http.MethodPost, path: refreshTokenPath,
			refreshToken: presentedRefreshToken}, wantStatus: http.StatusOK},
		{name: "token revoke", request: routeRequest{method: http.MethodPost, path: revokeTokenPath,
			refreshToken: presentedRefreshToken}, wantStatus: http.StatusOK},
		{name: "upload files", request: routeRequest{method: http.MethodPost, path: uploadFilesPath,
			body: uploadFilesBody, contentType: uploadContentType, authorization: bearerCredential},
			wantStatus: http.StatusCreated},
		{name: "create profile", request: routeRequest{method: http.MethodPost, path: profilesPath,
			body: profileBody, authorization: bearerCredential}, wantStatus: http.StatusCreated},
		{name: "update profile", request: routeRequest{method: http.MethodPut, path: profilesPath,
			body: profileBody, authorization: bearerCredential}, wantStatus: http.StatusOK},
	}

	router := routerWithEveryRouteWired()

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := serveRouteRequest(router, testCase.request)

			if recorder.Code != testCase.wantStatus {
				t.Errorf("%s %s = %d, want %d", testCase.request.method, testCase.request.path,
					recorder.Code, testCase.wantStatus)
			}
		})
	}
}

func TestBuildRouterRejectsUnroutedRequests(t *testing.T) {
	cases := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{name: "unknown path", method: http.MethodGet, path: "/api/nope", wantStatus: http.StatusNotFound},
		{name: "health rejects POST", method: http.MethodPost, path: HealthRoutePath,
			wantStatus: http.StatusMethodNotAllowed},
		{name: "signup rejects GET", method: http.MethodGet, path: signupPath,
			wantStatus: http.StatusMethodNotAllowed},
		{name: "signin rejects GET", method: http.MethodGet, path: signinPath,
			wantStatus: http.StatusMethodNotAllowed},
		{name: "verify email rejects GET", method: http.MethodGet, path: verifyEmailPath,
			wantStatus: http.StatusMethodNotAllowed},
		{name: "current user rejects POST", method: http.MethodPost, path: currentUserPath,
			wantStatus: http.StatusMethodNotAllowed},
		{name: "request email change rejects GET", method: http.MethodGet, path: requestEmailChangePath,
			wantStatus: http.StatusMethodNotAllowed},
		{name: "change password rejects GET", method: http.MethodGet, path: changePasswordPath,
			wantStatus: http.StatusMethodNotAllowed},
		{name: "token refresh rejects GET", method: http.MethodGet, path: refreshTokenPath,
			wantStatus: http.StatusMethodNotAllowed},
		{name: "token revoke rejects GET", method: http.MethodGet, path: revokeTokenPath,
			wantStatus: http.StatusMethodNotAllowed},
		{name: "upload files rejects GET", method: http.MethodGet, path: uploadFilesPath,
			wantStatus: http.StatusMethodNotAllowed},
		{name: "upload subpath", method: http.MethodPost, path: uploadFilesPath + "/extra",
			wantStatus: http.StatusNotFound},
		{name: "profiles rejects GET", method: http.MethodGet, path: profilesPath,
			wantStatus: http.StatusMethodNotAllowed},
		{name: "profiles subpath", method: http.MethodPut, path: profilesPath + "/extra",
			wantStatus: http.StatusNotFound},
		{name: "unversioned signup", method: http.MethodPost, path: "/api/users/signup",
			wantStatus: http.StatusNotFound},
		{name: "unversioned token refresh", method: http.MethodPost, path: "/api/tokens/refresh",
			wantStatus: http.StatusNotFound},
	}

	router := BuildRouter(RouterDependencies{})

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := serveRouteRequest(router, routeRequest{method: testCase.method, path: testCase.path})

			if recorder.Code != testCase.wantStatus {
				t.Errorf("%s %s = %d, want %d", testCase.method, testCase.path, recorder.Code, testCase.wantStatus)
			}
		})
	}
}

func TestBuildRouterRequiresAuthOnlyWhereDeclared(t *testing.T) {
	silenceRouteLogger(t)

	cases := []struct {
		name         string
		request      routeRequest
		requiresAuth bool
	}{
		{name: "health", request: routeRequest{method: http.MethodGet, path: HealthRoutePath}},
		{name: "signup", request: routeRequest{method: http.MethodPost, path: signupPath, body: signupBody}},
		{name: "signin", request: routeRequest{method: http.MethodPost, path: signinPath, body: signupBody}},
		{name: "verify email", request: routeRequest{method: http.MethodPost, path: verifyEmailPath},
			requiresAuth: true},
		{name: "delete user", request: routeRequest{method: http.MethodDelete, path: currentUserPath},
			requiresAuth: true},
		{name: "request email change", request: routeRequest{method: http.MethodPost, path: requestEmailChangePath,
			body: requestEmailChangeBody}, requiresAuth: true},
		{name: "change password", request: routeRequest{method: http.MethodPost, path: changePasswordPath,
			body: changePasswordBody}, requiresAuth: true},
		{name: "upload files", request: routeRequest{method: http.MethodPost, path: uploadFilesPath,
			body: uploadFilesBody, contentType: uploadContentType}, requiresAuth: true},
		{name: "create profile", request: routeRequest{method: http.MethodPost, path: profilesPath,
			body: profileBody}, requiresAuth: true},
		{name: "update profile", request: routeRequest{method: http.MethodPut, path: profilesPath,
			body: profileBody}, requiresAuth: true},
	}

	router := routerWithEveryRouteWired()

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := serveRouteRequest(router, testCase.request)

			rejected := recorder.Code == http.StatusUnauthorized
			if rejected != testCase.requiresAuth {
				t.Errorf("%s %s without a token = %d, want requiresAuth=%v", testCase.request.method,
					testCase.request.path, recorder.Code, testCase.requiresAuth)
			}
		})
	}
}
