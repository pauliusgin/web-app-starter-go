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

type signInUserGoogleRequest struct {
	IDToken string `json:"idToken"`
}

type signInUserGoogleResponse struct {
	readmodels.UserReadModel
	AccessToken string `json:"accessToken"`
}

// SignInUserGoogleUseCase is the domain call the google sign-in handler depends on.
type SignInUserGoogleUseCase func(
	ctx context.Context,
	params usecases.SignInUserGoogleParams,
) (usecases.SignInUserGoogleResult, error)

// SignInUserGoogleDependencies carries the google sign-in use case, the read model lookup and both token issuers.
type SignInUserGoogleDependencies struct {
	SignInUserGoogle     SignInUserGoogleUseCase
	GetUserReadModelByID GetUserReadModelByID
	IssueRefreshToken    IssueRefreshToken
	Jwt                  jwt.AccessTokenIssuer
}

// SignInUserGoogle returns a handler signing a user in with a google ID token, answering the user and access
// token and setting the refresh token cookie.
func SignInUserGoogle(d SignInUserGoogleDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		request, err := handlers.DecodeRequest[signInUserGoogleRequest](w, r)
		if err != nil {
			writeSignInGoogleFailure(w, err)
			return
		}

		googleIDTokenInput, err := input.ValidateGoogleIDTokenInput(input.GoogleIDTokenInput{IDToken: request.IDToken})
		if err != nil {
			writeSignInGoogleFailure(w, err)
			return
		}

		result, err := d.SignInUserGoogle(r.Context(), usecases.SignInUserGoogleParams{
			IDToken: googleIDTokenInput.IDToken,
		})
		if err != nil {
			writeSignInGoogleFailure(w, err)
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
			writeSignInGoogleFailure(w, err)
			return
		}

		userReadModel, err := d.GetUserReadModelByID(r.Context(), signedInUser.ID)
		if err != nil {
			writeSignInGoogleFailure(w, err)
			return
		}

		issuedTokens.setRefreshTokenCookie(w)
		handlers.WriteJSON(w, http.StatusOK, buildSignInUserGoogleResponse(userReadModel, issuedTokens.AccessToken))
	}
}

func buildSignInUserGoogleResponse(user readmodels.UserReadModel, accessToken string) signInUserGoogleResponse {
	return signInUserGoogleResponse{UserReadModel: user, AccessToken: accessToken}
}

var signInGoogleRules = []handlers.RejectionRule{
	handlers.RejectWhen[usererrors.ErrGoogleAccountNotFound](
		handlers.Rejection{Status: http.StatusNotFound, Code: handlers.ErrorCodeGoogleAccountNotFound}),
	handlers.RejectWhen[usererrors.ErrGoogleAccountNotLinked](
		handlers.Rejection{Status: http.StatusConflict, Code: handlers.ErrorCodeGoogleAccountNotLinked}),
}

func writeSignInGoogleFailure(w http.ResponseWriter, err error) {
	handlers.WriteRejection(w, handlers.WriteRejectionParams{
		Err:                   err,
		RuleSets:              [][]handlers.RejectionRule{handlers.RequestDecodingRules, googleIdentityRules, signInGoogleRules},
		UnexpectedFailureText: unexpectedGoogleSignInFailureText,
	})
}
