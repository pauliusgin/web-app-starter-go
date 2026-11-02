// Package tokens holds the HTTP handlers that rotate and revoke refresh tokens.
package tokens

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
)

type rotateRefreshTokenResponse struct {
	AccessToken string `json:"accessToken"`
}

// RotateRefreshTokenUseCase is the rotation call the refresh handler depends on.
type RotateRefreshTokenUseCase func(
	ctx context.Context,
	params refreshtoken.RotateParams,
) (refreshtoken.RotateResult, error)

// FindUserByID is the user lookup call the refresh handler depends on.
type FindUserByID func(ctx context.Context, id uuid.UUID) (models.User, error)

// RotateRefreshTokenDependencies carries the rotation use case, the token holder lookup and the access token issuer.
type RotateRefreshTokenDependencies struct {
	RotateRefreshToken RotateRefreshTokenUseCase
	FindUserByID       FindUserByID
	Jwt                jwt.AccessTokenIssuer
}

// RotateRefreshToken returns a handler trading the refresh token cookie for an access token and a rotated cookie.
func RotateRefreshToken(d RotateRefreshTokenDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		presentedToken, err := ReadRefreshTokenCookie(r)
		if err != nil {
			writeRefreshTokenFailure(w, err)
			return
		}

		rotated, err := d.RotateRefreshToken(r.Context(),
			refreshtoken.RotateParams{PresentedToken: presentedToken})
		if err != nil {
			writeRefreshTokenFailure(w, err)
			return
		}

		tokenHolder, err := d.FindUserByID(r.Context(), rotated.UserID)
		if err != nil {
			writeRefreshTokenFailure(w, err)
			return
		}

		accessToken, err := d.Jwt.IssueAccessToken(jwt.IssueAccessTokenParams{
			UserID: tokenHolder.ID,
			Email:  tokenHolder.Email,
		})
		if err != nil {
			writeRefreshTokenFailure(w, err)
			return
		}

		SetRefreshTokenCookie(w, SetRefreshTokenCookieParams{
			Plaintext: rotated.Plaintext,
			ExpiresAt: rotated.ExpiresAt,
		})
		handlers.WriteJSON(w, http.StatusOK, buildRotateRefreshTokenResponse(accessToken))
	}
}

func buildRotateRefreshTokenResponse(accessToken string) rotateRefreshTokenResponse {
	return rotateRefreshTokenResponse{AccessToken: accessToken}
}
