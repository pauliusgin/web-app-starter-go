package profiles

import (
	"context"

	"github.com/pauliusgin/web_app_starter-be/internal/domains/profiles/read/queries"
)

// GetProfileByIDUseCase is the read call the profile write handlers re-read the stored profile through.
type GetProfileByIDUseCase func(ctx context.Context, params queries.GetProfileByIDParams) (queries.GetProfileByIDResult, error)
