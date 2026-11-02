package users

import (
	"context"
	"net/http"

	"github.com/pauliusgin/web_app_starter-be/internal/api/http/handlers"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/read/queries"
)

// The client ID changes only on a manual rotation; after one, browsers keep the old ID until max-age runs out.
const googleSignInConfigCacheControl = "public, max-age=3600"

type getGoogleSignInConfigResponse struct {
	ClientID string `json:"clientId"`
}

// GetGoogleSignInConfigUseCase is the read-side lookup the google sign-in config handler depends on.
type GetGoogleSignInConfigUseCase func(ctx context.Context) (queries.GetGoogleSignInConfigResult, error)

// GetGoogleSignInConfigDependencies carries the query returning the google OAuth client ID.
type GetGoogleSignInConfigDependencies struct {
	GetGoogleSignInConfig GetGoogleSignInConfigUseCase
}

// GetGoogleSignInConfig returns a handler answering with the config a client renders the google button from.
func GetGoogleSignInConfig(d GetGoogleSignInConfigDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := d.GetGoogleSignInConfig(r.Context())
		if err != nil {
			handlers.WriteRejection(w, handlers.WriteRejectionParams{
				Err:                   err,
				UnexpectedFailureText: unexpectedGoogleSignInConfigFailureText,
			})

			return
		}

		w.Header().Set("Cache-Control", googleSignInConfigCacheControl)
		handlers.WriteJSON(w, http.StatusOK, buildGetGoogleSignInConfigResponse(result))
	}
}

func buildGetGoogleSignInConfigResponse(result queries.GetGoogleSignInConfigResult) getGoogleSignInConfigResponse {
	return getGoogleSignInConfigResponse{ClientID: result.ClientID}
}
