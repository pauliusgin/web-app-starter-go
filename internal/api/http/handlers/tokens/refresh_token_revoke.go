package tokens

import (
	"context"
	"errors"
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
)

// RevokeRefreshTokenUseCase is the logout call the revoke handler depends on.
type RevokeRefreshTokenUseCase func(
	ctx context.Context,
	params refreshtoken.RevokeParams,
) error

// RevokeRefreshTokenDependencies carries the collaborator the revoke handler orchestrates.
type RevokeRefreshTokenDependencies struct {
	RevokeRefreshToken RevokeRefreshTokenUseCase
}

// RevokeRefreshToken returns a handler ending the session the presented refresh token names.
func RevokeRefreshToken(d RevokeRefreshTokenDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		presentedToken, err := middleware.ReadBearerToken(r.Header.Get(authorizationHeader))
		if err != nil {
			w.WriteHeader(http.StatusOK)
			return
		}

		err = d.RevokeRefreshToken(r.Context(),
			refreshtoken.RevokeParams{PresentedToken: presentedToken})
		if err != nil && !errors.As(err, &refreshtoken.ErrRefreshTokenNotFound{}) {
			writeRefreshTokenErrorResponse(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
