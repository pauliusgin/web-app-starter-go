package users

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/tokens"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
)

var sessionTokensUserID = uuid.MustParse("6d1f4f0e-8f4b-4a4e-9c55-3f1f1f2d9a01")

func newSessionTokensParams(
	t *testing.T,
	refreshTokens *refreshtoken.InMemoryRefreshTokenRepository,
) issueSessionTokensParams {
	t.Helper()

	issuer, err := jwt.NewAccessTokenIssuer(jwt.NewAccessTokenIssuerParams{Secret: "session-tokens-test-secret"})
	if err != nil {
		t.Fatalf("new access token issuer: %v", err)
	}

	return issueSessionTokensParams{
		Jwt: issuer,
		IssueRefreshToken: func(ctx context.Context, params refreshtoken.IssueParams) (refreshtoken.IssueResult, error) {
			return refreshtoken.Issue(ctx, refreshtoken.IssueDependencies{WriteRepository: refreshTokens}, params)
		},
		UserID: sessionTokensUserID,
		Email:  testEmail,
	}
}

func TestIssueSessionTokensIssuesBothTokensForTheUser(t *testing.T) {
	refreshTokens := refreshtoken.NewInMemoryRefreshTokenRepository()
	params := newSessionTokensParams(t, refreshTokens)

	issuedTokens, err := issueSessionTokens(context.Background(), params)
	if err != nil {
		t.Fatalf("issue session tokens: %v", err)
	}

	claims, err := params.Jwt.ParseAccessToken(issuedTokens.AccessToken)
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	if claims.UserID != sessionTokensUserID {
		t.Errorf("access token user = %s, want %s", claims.UserID, sessionTokensUserID)
	}
	if issuedTokens.RefreshToken.Plaintext == "" {
		t.Error("refresh token plaintext is empty")
	}
	if len(refreshTokens.Tokens) != 1 {
		t.Fatalf("stored refresh tokens = %d, want 1", len(refreshTokens.Tokens))
	}
	for _, stored := range refreshTokens.Tokens {
		if stored.UserID != sessionTokensUserID {
			t.Errorf("stored refresh token user = %s, want %s", stored.UserID, sessionTokensUserID)
		}
	}
}

func TestIssueSessionTokensStoresNoRefreshTokenWhenTheAccessTokenFails(t *testing.T) {
	refreshTokens := refreshtoken.NewInMemoryRefreshTokenRepository()
	params := newSessionTokensParams(t, refreshTokens)
	params.Jwt = refusingAccessTokenIssuer{}

	_, err := issueSessionTokens(context.Background(), params)

	if !errors.As(err, &tokens.ErrAccessTokenIssuingFailed{}) {
		t.Fatalf("error = %v, want ErrAccessTokenIssuingFailed", err)
	}
	if len(refreshTokens.Tokens) != 0 {
		t.Errorf("stored refresh tokens = %d, want 0", len(refreshTokens.Tokens))
	}
}

func TestIssueSessionTokensReportsARefreshTokenFailure(t *testing.T) {
	refreshTokens := refreshtoken.NewInMemoryRefreshTokenRepository()
	refreshTokens.SaveFailure = errors.New("refresh_tokens table unavailable")
	params := newSessionTokensParams(t, refreshTokens)

	_, err := issueSessionTokens(context.Background(), params)

	if !errors.As(err, &tokens.ErrRefreshTokenIssuingFailed{}) {
		t.Fatalf("error = %v, want ErrRefreshTokenIssuingFailed", err)
	}
}
