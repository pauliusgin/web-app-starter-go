package users

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers/tokens"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/input"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	readmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/models"
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
	RefreshToken    string     `json:"refreshToken"`
}

// SignInUserUseCase is the domain call the sign-in handler depends on.
type SignInUserUseCase func(ctx context.Context, params usecases.SignInUserParams) (usecases.SignInUserResult, error)

// FindUserReadModelByID is the client-facing user lookup the sign-in handler answers with.
type FindUserReadModelByID func(ctx context.Context, id uuid.UUID) (readmodels.UserReadModel, error)

// SignInUserDependencies carries the collaborators the sign-in handler orchestrates.
type SignInUserDependencies struct {
	SignInUser            SignInUserUseCase
	FindUserReadModelByID FindUserReadModelByID
	IssueRefreshToken     IssueRefreshToken
	Jwt                   jwt.AccessTokenIssuer
}

// SignInUser returns a handler that checks credentials and answers with the user and tokens.
func SignInUser(d SignInUserDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request, err := handlers.DecodeRequest[signInUserRequest](w, r)
		if err != nil {
			writeSignInFailure(w, err)
			return
		}

		signInInput := input.SignInInput{
			Email:    normalizeEmail(request.Email),
			Password: request.Password,
		}
		if err := input.ValidateSignInInput(signInInput); err != nil {
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

		userReadModel, err := d.FindUserReadModelByID(r.Context(), signedInUser.ID)
		if err != nil {
			handlers.WriteUnexpectedFailure(w, handlers.WriteUnexpectedFailureParams{
				Err:         err,
				FailureText: unexpectedSignInFailureText,
			})
			return
		}

		accessToken, err := d.Jwt.IssueAccessToken(jwt.IssueAccessTokenParams{
			UserID: signedInUser.ID,
			Email:  signedInUser.Email,
		})
		if err != nil {
			writeSignInFailure(w, tokens.ErrAccessTokenIssuingFailed{Cause: err})
			return
		}

		refreshToken, err := d.IssueRefreshToken(r.Context(),
			refreshtoken.IssuingParams{UserID: signedInUser.ID})
		if err != nil {
			writeSignInFailure(w, tokens.ErrRefreshTokenIssuingFailed{Cause: err})
			return
		}

		handlers.WriteJSON(w, http.StatusOK, signInUserResponse{
			ID:              userReadModel.ID,
			Email:           userReadModel.Email,
			CreatedAt:       userReadModel.CreatedAt,
			EmailVerifiedAt: userReadModel.EmailVerifiedAt,
			SigninAt:        userReadModel.SigninAt,
			FirstName:       userReadModel.FirstName,
			LastName:        userReadModel.LastName,
			AccessToken:     accessToken,
			RefreshToken:    refreshToken.Plaintext,
		})
	}
}

var signInRules = []handlers.RejectionRule{
	handlers.RejectWhen[usererrors.ErrInvalidCredentials](
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
