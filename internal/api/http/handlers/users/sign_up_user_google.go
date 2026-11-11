package users

import (
	"context"
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/input"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/jwt"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	readmodels "github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/usecases"
)

type signUpUserGoogleRequest struct {
	IDToken string `json:"idToken"`
}

type signUpUserGoogleResponse struct {
	readmodels.UserReadModel
	AccessToken string `json:"accessToken"`
}

// SignUpUserGoogleUseCase is the domain call the google signup handler depends on.
type SignUpUserGoogleUseCase func(
	ctx context.Context,
	params usecases.SignUpUserGoogleParams,
) (usecases.SignUpUserGoogleResult, error)

// SignUpUserGoogleDependencies carries the google signup use case, the read model lookup and both token issuers.
type SignUpUserGoogleDependencies struct {
	SignUpUserGoogle     SignUpUserGoogleUseCase
	GetUserReadModelByID GetUserReadModelByID
	IssueRefreshToken    IssueRefreshToken
	Jwt                  jwt.AccessTokenIssuer
}

// SignUpUserGoogle returns a handler creating a user from a google ID token, answering it with an access token
// and setting the refresh token cookie.
func SignUpUserGoogle(d SignUpUserGoogleDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request, err := handlers.DecodeRequest[signUpUserGoogleRequest](w, r)
		if err != nil {
			writeSignUpGoogleFailure(w, err)
			return
		}

		googleIDTokenInput, err := input.ValidateGoogleIDTokenInput(input.GoogleIDTokenInput{IDToken: request.IDToken})
		if err != nil {
			writeSignUpGoogleFailure(w, err)
			return
		}

		result, err := d.SignUpUserGoogle(r.Context(), usecases.SignUpUserGoogleParams{
			IDToken: googleIDTokenInput.IDToken,
		})
		if err != nil {
			writeSignUpGoogleFailure(w, err)
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
			writeSignUpGoogleFailure(w, err)
			return
		}

		userReadModel, err := d.GetUserReadModelByID(r.Context(), signedUpUser.ID)
		if err != nil {
			writeSignUpGoogleFailure(w, err)
			return
		}

		issuedTokens.setRefreshTokenCookie(w)
		handlers.WriteJSON(w, http.StatusCreated, buildSignUpUserGoogleResponse(userReadModel, issuedTokens.AccessToken))
	}
}

func buildSignUpUserGoogleResponse(user readmodels.UserReadModel, accessToken string) signUpUserGoogleResponse {
	return signUpUserGoogleResponse{UserReadModel: user, AccessToken: accessToken}
}

var signUpGoogleRules = []handlers.RejectionRule{
	handlers.RejectWhen[usererrors.ErrGoogleAccountTaken](
		handlers.Rejection{Status: http.StatusConflict, Code: handlers.ErrorCodeGoogleAccountTaken}),
	handlers.RejectWhen[usererrors.ErrEmailTaken](
		handlers.Rejection{Status: http.StatusConflict, Code: handlers.ErrorCodeEmailTaken}),
}

func writeSignUpGoogleFailure(w http.ResponseWriter, err error) {
	handlers.WriteRejection(w, handlers.WriteRejectionParams{
		Err:                   err,
		RuleSets:              [][]handlers.RejectionRule{handlers.RequestDecodingRules, googleIdentityRules, signUpGoogleRules},
		UnexpectedFailureText: unexpectedGoogleSignUpFailureText,
	})
}
