package profiles

import (
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
	uploaderrors "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/errors"
)

var profileValidationRules = []handlers.RejectionRule{
	handlers.RejectWhen[primitive.ErrFirstNameMissing](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeFirstNameMissing}),
	handlers.RejectWhen[primitive.ErrLastNameMissing](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeLastNameMissing}),
	handlers.RejectWhen[primitive.ErrFirstNameInvalid](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeFirstNameInvalid}),
	handlers.RejectWhen[primitive.ErrLastNameInvalid](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeLastNameInvalid}),
	handlers.RejectWhen[primitive.ErrPictureUploadIDMalformed](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeProfilePictureUploadInvalid}),
}

// profilePictureRules answers a picture upload that is unknown, not the caller's or not a profile picture.
var profilePictureRules = []handlers.RejectionRule{
	handlers.RejectWhen[uploaderrors.ErrUploadNotFound](
		handlers.Rejection{Status: http.StatusBadRequest, Code: handlers.ErrorCodeProfilePictureUploadInvalid}),
}
