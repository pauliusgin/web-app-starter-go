package tokens

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
)

const (
	authorizationHeader   = "Authorization"
	wwwAuthenticateHeader = "WWW-Authenticate"
	invalidTokenChallenge = `Bearer error="invalid_token"`
)

const invalidRefreshTokenText = "invalid refresh token"
const refreshTokenRejectedText = "rejected a refresh token"
const unexpectedRefreshTokenFailureText = "could not complete the refresh token request"

func writeRefreshTokenErrorResponse(w http.ResponseWriter, err error) {
	if !refreshTokenRejected(err) {
		slog.Error(unexpectedRefreshTokenFailureText, "error", err)
		handlers.WriteJSON(w, http.StatusInternalServerError,
			handlers.ErrorResponse{Code: handlers.ErrorCodeInternal, Message: unexpectedRefreshTokenFailureText})

		return
	}

	slog.Warn(refreshTokenRejectedText, "error", err)
	w.Header().Set(wwwAuthenticateHeader, invalidTokenChallenge)
	handlers.WriteJSON(w, http.StatusUnauthorized, handlers.ErrorResponse{
		Code:    handlers.ErrorCodeRefreshTokenInvalid,
		Message: invalidRefreshTokenText,
	})
}

func refreshTokenRejected(err error) bool {
	rejections := []any{
		&middleware.ErrAuthorizationHeaderAbsent{},
		&middleware.ErrUnsupportedAuthorizationScheme{},
		&middleware.ErrBearerTokenMalformed{},
		&refreshtoken.ErrRefreshTokenNotFound{},
		&usererrors.ErrUserNotFound{},
	}

	for _, rejection := range rejections {
		if errors.As(err, rejection) {
			return true
		}
	}
	return false
}
