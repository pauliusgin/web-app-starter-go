package profiles

import (
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
)

var profileValidationRules = []handlers.RejectionRule{
	handlers.RejectWhen[primitive.ErrFirstNameMissing](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeFirstNameMissing}),
	handlers.RejectWhen[primitive.ErrLastNameMissing](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeLastNameMissing}),
}
