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
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

type signInUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type signInUserResponse struct {
	ID              uuid.UUID  `json:"id"`
	Email           string     `json:"email"`
	CreatedAt       time.Time  `json:"createdAt"`
	EmailVerifiedAt *time.Time `json:"emailVerifiedAt"`
	SigninAt        *time.Time `json:"signinAt"`
	FirstName       *string    `json:"firstName"`
	LastName        *string    `json:"lastName"`
	AccessToken     string     `json:"accessToken"`
}

// SignInUserUseCase is the domain call the sign-in handler depends on.
type SignInUserUseCase func(ctx context.Context, params usecases.SignInUserParams) (usecases.SignInUserResult, error)

// GetUserReadModelByID is the client-facing user lookup the sign-in handler answers with.
type GetUserReadModelByID func(ctx context.Context, id uuid.UUID) (models.UserReadModel, error)

// SignInUserDependencies carries the credentials check, the read model lookup it answers with, and both token issuers.
type SignInUserDependencies struct {
	SignInUser           SignInUserUseCase
	GetUserReadModelByID GetUserReadModelByID
	IssueRefreshToken    IssueRefreshToken
	Jwt                  jwt.AccessTokenIssuer
}

// SignInUser returns a handler checking credentials, answering the user and access token, setting the refresh cookie.
func SignInUser(d SignInUserDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request, err := handlers.DecodeRequest[signInUserRequest](w, r)
		if err != nil {
			writeSignInFailure(w, err)
			return
		}

		signInInput, err := input.ValidateSignInInput(input.SignInInput{
			Email:    request.Email,
			Password: request.Password,
		})
		if err != nil {
			writeSignInFailure(w, err)
			return
		}

		result, err := d.SignInUser(r.Context(), usecases.SignInUserParams{
			Email:    signInInput.Email,
			Password: signInInput.Password,
		})
		if err != nil {
			writeSignInFailure(w, err)
			return
		}

		signedInUser := result.User

		issuedTokens, err := issueSessionTokens(r.Context(), issueSessionTokensParams{
			Jwt:               d.Jwt,
			IssueRefreshToken: d.IssueRefreshToken,
			UserID:            signedInUser.ID,
			Email:             signedInUser.Email,
		})
		if err != nil {
			writeSignInFailure(w, err)
			return
		}

		userReadModel, err := d.GetUserReadModelByID(r.Context(), signedInUser.ID)
		if err != nil {
			writeSignInFailure(w, err)
			return
		}

		issuedTokens.setRefreshTokenCookie(w)
		handlers.WriteJSON(w, http.StatusOK, buildSignInUserResponse(userReadModel, issuedTokens.AccessToken))
	}
}

func buildSignInUserResponse(user models.UserReadModel, accessToken string) signInUserResponse {
	return signInUserResponse{
		ID:              user.ID,
		Email:           user.Email,
		CreatedAt:       user.CreatedAt,
		EmailVerifiedAt: user.EmailVerifiedAt,
		SigninAt:        user.SigninAt,
		FirstName:       user.FirstName,
		LastName:        user.LastName,
		AccessToken:     accessToken,
	}
}

var signInRules = []handlers.RejectionRule{
	handlers.RejectWhen[usererrors.ErrCredentialsInvalid](
		handlers.Rejection{Status: http.StatusUnauthorized, Code: handlers.ErrorCodeInvalidCredentials}),
}

func writeSignInFailure(w http.ResponseWriter, err error) {
	handlers.WriteRejection(w, handlers.WriteRejectionParams{
		Err: err,
		RuleSets: [][]handlers.RejectionRule{
			handlers.RequestDecodingRules, emailValidationRules, passwordValidationRules, signInRules,
		},
		UnexpectedFailureText: unexpectedSignInFailureText,
	})
}
