// Package users holds the HTTP handlers for signing up, signing in, and verifying or changing an email.
package users

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/input"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	readmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

type signUpUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type signUpUserResponse struct {
	ID              uuid.UUID                 `json:"id"`
	Email           string                    `json:"email"`
	CreatedAt       time.Time                 `json:"createdAt"`
	EmailVerifiedAt *time.Time                `json:"emailVerifiedAt"`
	SigninAt        *time.Time                `json:"signinAt"`
	FirstName       *string                   `json:"firstName"`
	LastName        *string                   `json:"lastName"`
	SignInMethods   []readmodels.SignInMethod `json:"signInMethods"`
	AccessToken     string                    `json:"accessToken"`
}

// SignUpUserUseCase is the domain call the signup handler depends on.
type SignUpUserUseCase func(ctx context.Context, params usecases.SignUpUserParams) (usecases.SignUpUserResult, error)

// SignUpUserDependencies carries the sign-up use case, the read model lookup it answers with, and both token issuers.
type SignUpUserDependencies struct {
	SignUpUser           SignUpUserUseCase
	GetUserReadModelByID GetUserReadModelByID
	IssueRefreshToken    IssueRefreshToken
	Jwt                  jwt.AccessTokenIssuer
}

// SignUpUser returns a handler that creates a user and answers with an access token, setting the refresh token cookie.
func SignUpUser(d SignUpUserDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request, err := handlers.DecodeRequest[signUpUserRequest](w, r)
		if err != nil {
			writeSignUpFailure(w, err)
			return
		}

		signUpInput, err := input.ValidateSignUpInput(input.SignUpInput{
			Email:    request.Email,
			Password: request.Password,
		})
		if err != nil {
			writeSignUpFailure(w, err)
			return
		}

		result, err := d.SignUpUser(r.Context(), usecases.SignUpUserParams{
			Email:    signUpInput.Email,
			Password: signUpInput.Password,
		})
		if err != nil {
			writeSignUpFailure(w, err)
			return
		}

		signedUpUser := result.User

		issuedTokens, err := issueSessionTokens(r.Context(), issueSessionTokensParams{
			Jwt:               d.Jwt,
			IssueRefreshToken: d.IssueRefreshToken,
			UserID:            signedUpUser.ID,
			Email:             signedUpUser.Email,
		})
		if err != nil {
			writeSignUpFailure(w, err)
			return
		}

		userReadModel, err := d.GetUserReadModelByID(r.Context(), signedUpUser.ID)
		if err != nil {
			writeSignUpFailure(w, err)
			return
		}

		issuedTokens.setRefreshTokenCookie(w)
		handlers.WriteJSON(w, http.StatusCreated, buildSignUpUserResponse(userReadModel, issuedTokens.AccessToken))
	}
}

func buildSignUpUserResponse(user readmodels.UserReadModel, accessToken string) signUpUserResponse {
	return signUpUserResponse{
		ID:              user.ID,
		Email:           user.Email,
		CreatedAt:       user.CreatedAt,
		EmailVerifiedAt: user.EmailVerifiedAt,
		SigninAt:        user.SigninAt,
		FirstName:       user.FirstName,
		LastName:        user.LastName,
		SignInMethods:   user.SignInMethods,
		AccessToken:     accessToken,
	}
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
		UnexpectedFailureText: unexpectedSignUpFailureText,
	})
}
