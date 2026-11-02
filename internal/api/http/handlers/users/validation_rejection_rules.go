package users

import (
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/googleidtoken"
	usererrors "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
)

var emailValidationRules = []handlers.RejectionRule{
	handlers.RejectWhen[primitive.ErrEmailInvalid](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeEmailInvalid}),
}

var passwordValidationRules = []handlers.RejectionRule{
	handlers.RejectWhen[primitive.ErrPasswordMissing](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodePasswordMissing}),
	handlers.RejectWhen[primitive.ErrPasswordTooShort](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodePasswordTooShort}),
	handlers.RejectWhen[primitive.ErrPasswordTooLong](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodePasswordTooLong}),
	handlers.RejectWhen[primitive.ErrPasswordContainsWhitespace](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodePasswordContainsWhitespace}),
}

// googleIdentityRules reject an ID token before any account is looked at, shared by google signup and sign-in.
var googleIdentityRules = []handlers.RejectionRule{
	handlers.RejectWhen[primitive.ErrGoogleIDTokenMissing](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeGoogleIDTokenMissing}),
	handlers.RejectWhen[googleidtoken.ErrIDTokenInvalid](
		handlers.Rejection{Status: http.StatusUnauthorized, Code: handlers.ErrorCodeGoogleIDTokenInvalid}),
	handlers.RejectWhen[usererrors.ErrGoogleEmailUnverified](
		handlers.Rejection{Status: http.StatusForbidden, Code: handlers.ErrorCodeGoogleEmailUnverified}),
}
