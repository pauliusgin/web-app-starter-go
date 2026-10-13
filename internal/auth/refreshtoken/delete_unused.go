package refreshtoken

import "context"

// UnusedDeletionDependencies carries the collaborator purging dead tokens needs.
type UnusedDeletionDependencies struct {
	WriteRepository WriteRepository
}

// UnusedDeletionParams are the parameters for the unused token deletion.
type UnusedDeletionParams struct {
	CleanUpThreshold int32
}

// DeleteUnusedRefreshTokens purges expired and long-revoked tokens and returns the row count.
func DeleteUnusedRefreshTokens(
	ctx context.Context,
	d UnusedDeletionDependencies,
	params UnusedDeletionParams,
) (int64, error) {
	return d.WriteRepository.DeleteUnused(ctx, params.CleanUpThreshold)
}
