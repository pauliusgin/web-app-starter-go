package refreshtoken

import (
	"context"

	"github.com/google/uuid"
)

// DeleteUserRefreshTokensDependencies carries the collaborator deleting a user's tokens needs.
type DeleteUserRefreshTokensDependencies struct {
	WriteRepository WriteRepository
}

// DeleteUserRefreshTokensParams is the user whose refresh tokens are being deleted.
type DeleteUserRefreshTokensParams struct {
	UserID uuid.UUID
}

// DeleteUserRefreshTokens hard-deletes every refresh token the user holds, ending all sessions.
func DeleteUserRefreshTokens(
	ctx context.Context,
	d DeleteUserRefreshTokensDependencies,
	params DeleteUserRefreshTokensParams,
) error {
	return d.WriteRepository.DeleteAllForUser(ctx, params.UserID)
}
