// Package users holds the HTTP handlers for signing up, signing in, and verifying or changing an email.
package users

import (
	"context"
	"net/http"
	"strings"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/tokens"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/input"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

type signUpUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type signUpUserResponse struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// SignupUser is the domain call the signup handler depends on.
type SignupUser func(ctx context.Context, params usecases.SignupUserParams) (usecases.SignupUserResult, error)

// IssueRefreshToken is the refresh token call the signup handler depends on.
type IssueRefreshToken func(
	ctx context.Context,
	params refreshtoken.IssuingParams,
) (refreshtoken.Issued, error)

// SignUpUserDependencies carries the collaborators the signup handler orchestrates.
type SignUpUserDependencies struct {
	SignupUser        SignupUser
	IssueRefreshToken IssueRefreshToken
	Jwt               jwt.AccessTokenIssuer
}

// SignUpUser returns a handler that creates a user and answers with the pair of session tokens.
func SignUpUser(d SignUpUserDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request, err := handlers.DecodeRequest[signUpUserRequest](w, r)
		if err != nil {
			writeSignUpFailure(w, err)
			return
		}

		signUpInput := input.SignUpInput{
			Email:    normalizeEmail(request.Email),
			Password: request.Password,
		}
		if err := input.ValidateSignUpInput(signUpInput); err != nil {
			writeSignUpFailure(w, err)
			return
		}

		result, err := d.SignupUser(r.Context(), usecases.SignupUserParams{
			Email:    signUpInput.Email,
			Password: signUpInput.Password,
		})
		if err != nil {
			writeSignUpFailure(w, err)
			return
		}

		signedUpUser := result.User

		accessToken, err := d.Jwt.IssueAccessToken(jwt.IssueAccessTokenParams{
			UserID: signedUpUser.ID,
			Email:  signedUpUser.Email,
		})
		if err != nil {
			writeSignUpFailure(w, tokens.ErrAccessTokenIssuingFailed{Cause: err})
			return
		}

		refreshToken, err := d.IssueRefreshToken(r.Context(),
			refreshtoken.IssuingParams{UserID: signedUpUser.ID})
		if err != nil {
			writeSignUpFailure(w, tokens.ErrRefreshTokenIssuingFailed{Cause: err})
			return
		}

		handlers.WriteJSON(w, http.StatusCreated, signUpUserResponse{
			ID:           signedUpUser.ID.String(),
			Email:        signedUpUser.Email,
			AccessToken:  accessToken,
			RefreshToken: refreshToken.Plaintext,
		})
	}
}

func normalizeEmail(rawEmail string) string {
	return strings.ToLower(strings.TrimSpace(rawEmail))
}

var signUpRules = []handlers.RejectionRule{
	handlers.RejectWhen[usererrors.ErrEmailTaken](
		handlers.Rejection{Status: http.StatusConflict, Code: handlers.ErrorCodeEmailTaken}),
}

func writeSignUpFailure(w http.ResponseWriter, err error) {
	handlers.WriteRejection(w, handlers.WriteRejectionParams{
		Err: err,
		RuleSets: [][]handlers.RejectionRule{
			handlers.RequestDecodingRules, emailValidationRules, passwordValidationRules, signUpRules,
		},
		UnexpectedFailureText: unexpectedSignupFailureText,
	})
}
