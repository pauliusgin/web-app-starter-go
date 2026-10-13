package users

import (
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
)

var emailValidationRules = []handlers.RejectionRule{
	handlers.RejectWhen[primitive.ErrInvalidEmail](
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
