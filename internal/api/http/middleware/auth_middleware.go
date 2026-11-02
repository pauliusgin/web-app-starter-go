// Package middleware holds the HTTP filters that run around the API's handlers.
package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
)

const (
	authorizationHeader = "Authorization"
	bearerScheme        = "Bearer"
)

const (
	tokenRequiredChallenge  = "Bearer"
	expiredTokenChallenge   = `Bearer error="invalid_token", error_description="the access token expired"`
	accessTokenRequiredText = "an access token is required"
	accessTokenExpiredText  = "the access token expired"
	invalidAccessTokenText  = "invalid access token"
)

const unexpectedAuthenticationFailureText = "could not authenticate the request"

// ParseAccessToken is the token parsing call the authentication middleware depends on.
type ParseAccessToken func(accessToken string) (jwt.AccessTokenClaims, error)

// FindUserByID is the user lookup call the authentication middleware depends on.
type FindUserByID func(ctx context.Context, id uuid.UUID) (models.User, error)

// RequireAuthenticatedUserDependencies carries the access token parser and the lookup confirming the user still exists.
type RequireAuthenticatedUserDependencies struct {
	ParseAccessToken ParseAccessToken
	FindUserByID     FindUserByID
}

// RequireAuthenticatedUser returns middleware admitting only tokens naming a live user.
func RequireAuthenticatedUser(d RequireAuthenticatedUserDependencies) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			accessToken, err := ReadBearerToken(r.Header.Get(authorizationHeader))
			if err != nil {
				writeUnauthorized(w, err)
				return
			}

			claims, err := d.ParseAccessToken(accessToken)
			if err != nil {
				writeUnauthorized(w, ErrAccessTokenRejected{Cause: err})
				return
			}

			if _, err := d.FindUserByID(r.Context(), claims.UserID); err != nil {
				writeUserLookupFailure(w, writeUserLookupFailureParams{UserID: claims.UserID, Err: err})
				return
			}

			authenticated := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(authenticated))
		})
	}
}

// ReadBearerToken returns the credential an Authorization header carries, or why it was rejected.
func ReadBearerToken(rawHeader string) (string, error) {
	if rawHeader == "" {
		return "", ErrAuthorizationHeaderMissing{}
	}

	scheme, accessToken, _ := strings.Cut(rawHeader, " ")
	if !strings.EqualFold(scheme, bearerScheme) {
		return "", ErrUnsupportedAuthorizationScheme{Scheme: truncateScheme(scheme)}
	}
	if accessToken == "" || strings.ContainsAny(accessToken, " \t") {
		return "", ErrBearerTokenMalformed{}
	}

	return accessToken, nil
}

const maxLoggedSchemeLength = 6

func truncateScheme(scheme string) string {
	schemeRunes := []rune(scheme)
	if len(schemeRunes) <= maxLoggedSchemeLength {
		return scheme
	}

	return string(schemeRunes[:maxLoggedSchemeLength]) + "..."
}

type writeUserLookupFailureParams struct {
	UserID uuid.UUID
	Err    error
}

func writeUserLookupFailure(w http.ResponseWriter, params writeUserLookupFailureParams) {
	userNotFound := usererrors.ErrUserNotFound{}
	if errors.As(params.Err, &userNotFound) {
		writeUnauthorized(w, ErrAuthenticatedUserNotFound{UserID: params.UserID, Cause: params.Err})
		return
	}

	slog.Error(unexpectedAuthenticationFailureText, "error", params.Err)
	handlers.WriteJSON(w, http.StatusInternalServerError, handlers.ErrorResponse{
		Code:    handlers.ErrorCodeInternal,
		Message: unexpectedAuthenticationFailureText,
	})
}

func writeUnauthorized(w http.ResponseWriter, err error) {
	challenge := selectUnauthorizedChallenge(err)
	w.Header().Set(handlers.WWWAuthenticateHeader, challenge.header)
	handlers.WriteJSON(w, http.StatusUnauthorized, handlers.ErrorResponse{Code: challenge.code, Message: challenge.body})
}

type unauthorizedChallenge struct {
	header string
	code   handlers.ErrorCode
	body   string
}

func selectUnauthorizedChallenge(err error) unauthorizedChallenge {
	headerMissing := ErrAuthorizationHeaderMissing{}
	tokenExpired := jwt.ErrTokenExpired{}

	switch {
	case errors.As(err, &headerMissing):
		return unauthorizedChallenge{
			header: tokenRequiredChallenge,
			code:   handlers.ErrorCodeAccessTokenMissing,
			body:   accessTokenRequiredText,
		}
	case errors.As(err, &tokenExpired):
		return unauthorizedChallenge{
			header: expiredTokenChallenge,
			code:   handlers.ErrorCodeAccessTokenExpired,
			body:   accessTokenExpiredText,
		}
	default:
		return unauthorizedChallenge{
			header: handlers.InvalidTokenChallenge,
			code:   handlers.ErrorCodeAccessTokenInvalid,
			body:   invalidAccessTokenText,
		}
	}
}

type contextKey struct{}

var userIDContextKey contextKey

// UserIDFromContext returns the user id the middleware stored, and false when there is none.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDContextKey).(uuid.UUID)

	return userID, ok
}
