package users

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/tokens"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
)

// IssueRefreshToken is the refresh token call the signup and sign-in handlers depend on.
type IssueRefreshToken func(
	ctx context.Context,
	params refreshtoken.IssueParams,
) (refreshtoken.IssueResult, error)

type sessionTokens struct {
	AccessToken  string
	RefreshToken refreshtoken.IssueResult
}

type issueSessionTokensParams struct {
	Jwt               jwt.AccessTokenIssuer
	IssueRefreshToken IssueRefreshToken
	UserID            uuid.UUID
	Email             string
}

func issueSessionTokens(ctx context.Context, params issueSessionTokensParams) (sessionTokens, error) {
	accessToken, err := params.Jwt.IssueAccessToken(jwt.IssueAccessTokenParams{
		UserID: params.UserID,
		Email:  params.Email,
	})
	if err != nil {
		return sessionTokens{}, tokens.ErrAccessTokenIssuingFailed{Cause: err}
	}

	refreshToken, err := params.IssueRefreshToken(ctx, refreshtoken.IssueParams{UserID: params.UserID})
	if err != nil {
		return sessionTokens{}, tokens.ErrRefreshTokenIssuingFailed{Cause: err}
	}

	return sessionTokens{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (s sessionTokens) setRefreshTokenCookie(w http.ResponseWriter) {
	tokens.SetRefreshTokenCookie(w, tokens.SetRefreshTokenCookieParams{
		Plaintext: s.RefreshToken.Plaintext,
		ExpiresAt: s.RefreshToken.ExpiresAt,
	})
}
