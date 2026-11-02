package tokens

import (
	"context"
	"errors"
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
)

// RevokeRefreshTokenUseCase is the logout call the revoke handler depends on.
type RevokeRefreshTokenUseCase func(
	ctx context.Context,
	params refreshtoken.RevokeParams,
) error

// RevokeRefreshTokenDependencies carries the use case revoking the presented refresh token.
type RevokeRefreshTokenDependencies struct {
	RevokeRefreshToken RevokeRefreshTokenUseCase
}

// RevokeRefreshToken returns a handler ending the session the refresh token cookie names and clearing the cookie.
func RevokeRefreshToken(d RevokeRefreshTokenDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		presentedToken, err := ReadRefreshTokenCookie(r)
		ClearRefreshTokenCookie(w)
		if err != nil {
			w.WriteHeader(http.StatusOK)
			return
		}

		err = d.RevokeRefreshToken(r.Context(),
			refreshtoken.RevokeParams{PresentedToken: presentedToken})
		if err != nil && !errors.As(err, &refreshtoken.ErrRefreshTokenNotFound{}) {
			writeRefreshTokenFailure(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
