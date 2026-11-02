package tokens

import (
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
)

const (
	wwwAuthenticateHeader = "WWW-Authenticate"
	invalidTokenChallenge = `Bearer error="invalid_token"`
)

const invalidRefreshTokenText = "invalid refresh token"
const refreshTokenRejectedText = "rejected a refresh token"
const unexpectedRefreshTokenFailureText = "could not complete the refresh token request"

// One code for every refresh token rejection, so a client cannot tell a missing cookie, a revoked token
// and a deleted account apart.
var refreshTokenInvalidRejection = handlers.Rejection{
	Status:  http.StatusUnauthorized,
	Code:    handlers.ErrorCodeRefreshTokenInvalid,
	Message: invalidRefreshTokenText,
}

var refreshTokenRules = []handlers.RejectionRule{
	handlers.RejectWhen[ErrRefreshTokenCookieMissing](refreshTokenInvalidRejection),
	handlers.RejectWhen[refreshtoken.ErrRefreshTokenNotFound](refreshTokenInvalidRejection),
	handlers.RejectWhen[usererrors.ErrUserNotFound](refreshTokenInvalidRejection),
}

func writeRefreshTokenFailure(w http.ResponseWriter, err error) {
	if _, rejected := handlers.SelectRejection(err, refreshTokenRules); rejected {
		ClearRefreshTokenCookie(w)
		w.Header().Set(wwwAuthenticateHeader, invalidTokenChallenge)
	}

	handlers.WriteRejection(w, handlers.WriteRejectionParams{
		Err:                   err,
		RuleSets:              [][]handlers.RejectionRule{refreshTokenRules},
		UnexpectedFailureText: unexpectedRefreshTokenFailureText,
		RejectedLogText:       refreshTokenRejectedText,
	})
}
