// Package tokens holds the HTTP handlers that rotate and revoke refresh tokens.
package tokens

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/middleware"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
)

type rotateRefreshTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// RotateRefreshTokenUseCase is the rotation call the refresh handler depends on.
type RotateRefreshTokenUseCase func(
	ctx context.Context,
	params refreshtoken.RotateParams,
) (refreshtoken.Rotated, error)

// FindUserByID is the user lookup call the refresh handler depends on.
type FindUserByID func(ctx context.Context, id uuid.UUID) (models.User, error)

// RotateDependencies carries the collaborators the refresh handler orchestrates.
type RotateDependencies struct {
	RotateRefreshToken RotateRefreshTokenUseCase
	FindUserByID       FindUserByID
	Jwt                jwt.AccessTokenIssuer
}

// RotateRefreshToken returns a handler exchanging a refresh token for a fresh pair of tokens.
func RotateRefreshToken(d RotateDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		presentedToken, err := middleware.ReadBearerToken(r.Header.Get(authorizationHeader))
		if err != nil {
			writeRefreshTokenErrorResponse(w, err)
			return
		}

		rotated, err := d.RotateRefreshToken(r.Context(),
			refreshtoken.RotateParams{PresentedToken: presentedToken})
		if err != nil {
			writeRefreshTokenErrorResponse(w, err)
			return
		}

		tokenHolder, err := d.FindUserByID(r.Context(), rotated.UserID)
		if err != nil {
			writeRefreshTokenErrorResponse(w, err)
			return
		}

		accessToken, err := d.Jwt.IssueAccessToken(jwt.IssueAccessTokenParams{
			UserID: tokenHolder.ID,
			Email:  tokenHolder.Email,
		})
		if err != nil {
			writeRefreshTokenErrorResponse(w, err)
			return
		}

		handlers.WriteJSON(w, http.StatusOK, rotateRefreshTokenResponse{
			AccessToken:  accessToken,
			RefreshToken: rotated.Plaintext,
		})
	}
}
