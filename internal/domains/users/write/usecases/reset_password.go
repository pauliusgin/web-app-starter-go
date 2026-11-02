package usecases

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/password"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	userevents "github.com/pauliusgin/web_app_starter-be/internal/events/user"
)

// ResetPasswordParams is the token the reset link carried and the password the user picked.
type ResetPasswordParams struct {
	Token       string
	NewPassword string
}

// DeleteUserRefreshTokens is the session-ending call resetting a password depends on.
type DeleteUserRefreshTokens func(ctx context.Context, params refreshtoken.DeleteUserRefreshTokensParams) error

// ResetPasswordDependencies carries the user store, the dispatcher, the challenge lookup and spend calls and
// the call ending the user's sessions.
type ResetPasswordDependencies struct {
	UserWriteRepository          repositories.UserRepository
	EventDispatcher              events.Dispatcher
	FindAuthChallengeByTokenHash FindAuthChallengeByTokenHash
	ConsumeAuthChallenge         ConsumeAuthChallenge
	DeleteUserRefreshTokens      DeleteUserRefreshTokens
}

// ResetPassword stores the new password of the account the reset link was mailed to, marks its email
// verified since the link reached the inbox, spends the link, ends every session and announces the reset.
func ResetPassword(ctx context.Context, d ResetPasswordDependencies, params ResetPasswordParams) error {
	resetLink, err := findPasswordResetLink(ctx, d.FindAuthChallengeByTokenHash, params.Token)
	if err != nil {
		return err
	}

	credentials, err := findCredentialsForResetLink(ctx, d.UserWriteRepository, resetLink)
	if err != nil {
		return err
	}

	err = storeResetPassword(ctx, d.UserWriteRepository, storeResetPasswordParams{
		UserID:      credentials.ID,
		NewPassword: params.NewPassword,
	})
	if err != nil {
		return err
	}

	if err := markEmailVerifiedByResetLink(ctx, d, credentials.ID); err != nil {
		return err
	}

	err = revokePreResetAccess(ctx, d, revokePreResetAccessParams{
		ResetLinkID: resetLink.ID,
		UserID:      credentials.ID,
	})
	if err != nil {
		return err
	}

	userevents.DispatchUserPasswordReset(ctx, d.EventDispatcher, userevents.NewUserPasswordResetParams{
		UserID: credentials.ID,
	})

	return nil
}

func findPasswordResetLink(
	ctx context.Context,
	findAuthChallengeByTokenHash FindAuthChallengeByTokenHash,
	presentedToken string,
) (challenge.Challenge, error) {
	matched, err := findAuthChallengeByTokenHash(ctx, token.Hash(presentedToken))
	if errors.As(err, &challenge.ErrChallengeNotFound{}) {
		return challenge.Challenge{}, e.ErrPasswordResetLinkInvalid{}
	}
	if err != nil {
		return challenge.Challenge{}, err
	}

	if matched.Purpose != challenge.PurposePasswordReset {
		return challenge.Challenge{}, e.ErrPasswordResetLinkInvalid{}
	}

	return matched, nil
}

// findCredentialsForResetLink refuses a link mailed to an address the account no longer uses: whoever
// held the old inbox must not take over the account.
func findCredentialsForResetLink(
	ctx context.Context,
	userWriteRepository repositories.UserRepository,
	resetLink challenge.Challenge,
) (models.UserCredentials, error) {
	credentials, err := userWriteRepository.FindUserCredentialsByID(ctx, resetLink.UserID)
	if errors.As(err, &e.ErrUserNotFound{}) {
		return models.UserCredentials{}, e.ErrPasswordResetLinkInvalid{}
	}
	if err != nil {
		return models.UserCredentials{}, err
	}

	if credentials.Email != resetLink.Target || !credentials.HasPassword() {
		return models.UserCredentials{}, e.ErrPasswordResetLinkInvalid{}
	}

	return credentials, nil
}

type storeResetPasswordParams struct {
	UserID      uuid.UUID
	NewPassword string
}

func storeResetPassword(
	ctx context.Context,
	userWriteRepository repositories.UserRepository,
	params storeResetPasswordParams,
) error {
	newPasswordHash, err := password.Hash(params.NewPassword)
	if err != nil {
		return e.ErrPasswordHashingFailed{Cause: err}
	}

	return userWriteRepository.ChangePassword(ctx, models.ChangePasswordParams{
		ID:           params.UserID,
		PasswordHash: newPasswordHash,
	})
}

func markEmailVerifiedByResetLink(ctx context.Context, d ResetPasswordDependencies, userID uuid.UUID) error {
	storedUser, err := d.UserWriteRepository.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if storedUser.IsEmailVerified() {
		return nil
	}

	verified, err := d.UserWriteRepository.MarkEmailVerified(ctx, userID)
	if err != nil {
		return err
	}

	userevents.DispatchUserEmailVerified(ctx, d.EventDispatcher, userevents.NewUserEmailVerifiedParams{
		UserID: verified.ID,
		Email:  verified.Email,
	})

	return nil
}

type revokePreResetAccessParams struct {
	ResetLinkID uuid.UUID
	UserID      uuid.UUID
}

// revokePreResetAccess spends the link and ends every session, so neither a replayed link nor a session
// opened with the old password outlives the reset.
func revokePreResetAccess(ctx context.Context, d ResetPasswordDependencies, params revokePreResetAccessParams) error {
	if err := d.ConsumeAuthChallenge(ctx, params.ResetLinkID); err != nil {
		return err
	}

	return d.DeleteUserRefreshTokens(ctx, refreshtoken.DeleteUserRefreshTokensParams{UserID: params.UserID})
}
