package handlers

import (
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/validators/primitive"
)

// PageRequestRejectionRules answers the page and limit query parameters every paginated list shares.
var PageRequestRejectionRules = []RejectionRule{
	RejectWhen[primitive.ErrPageNumberInvalid](Rejection{Status: http.StatusBadRequest, Code: ErrorCodePageInvalid}),
	RejectWhen[primitive.ErrPageLimitInvalid](Rejection{Status: http.StatusBadRequest, Code: ErrorCodeLimitInvalid}),
}
