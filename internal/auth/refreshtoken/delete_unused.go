package refreshtoken

import "context"

// DeleteUnusedRefreshTokensDependencies carries the refresh token store the purge runs against.
type DeleteUnusedRefreshTokensDependencies struct {
	WriteRepository WriteRepository
}

// DeleteUnusedRefreshTokensParams says how many days a revoked token outlives its revocation before it is purged.
type DeleteUnusedRefreshTokensParams struct {
	CleanUpThresholdInDays int32
}

// DeleteUnusedRefreshTokens purges expired and long-revoked tokens and returns the row count.
func DeleteUnusedRefreshTokens(
	ctx context.Context,
	d DeleteUnusedRefreshTokensDependencies,
	params DeleteUnusedRefreshTokensParams,
) (int64, error) {
	return d.WriteRepository.DeleteUnused(ctx, params.CleanUpThresholdInDays)
}
