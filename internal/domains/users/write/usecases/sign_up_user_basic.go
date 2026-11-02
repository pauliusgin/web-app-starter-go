// Package usecases holds the user write use cases.
package usecases

import (
	"context"
	"errors"
	"log/slog"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/password"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	userevents "github.com/pauliusgin/web_app_starter-be/internal/events/user"
)

// SignUpUserParams is the plaintext signup input; the password is hashed before storage.
type SignUpUserParams struct {
	Email    string
	Password string
}

// SignUpUserResult is the user the signup stored.
type SignUpUserResult struct {
	User models.User
}

// GenerateAuthChallenge is the challenge call creating a user depends on.
type GenerateAuthChallenge func(
	ctx context.Context,
	params challenge.GenerateParams,
) (challenge.GenerateResult, error)

// SignUpUserDependencies carries the user store, the event dispatcher and the verification link generator.
type SignUpUserDependencies struct {
	UserWriteRepository   repositories.UserRepository
	EventDispatcher       events.Dispatcher
	GenerateAuthChallenge GenerateAuthChallenge
}

// SignUpUser hashes the password, stores the user, announces the signup and asks for verification.
func SignUpUser(
	ctx context.Context,
	d SignUpUserDependencies,
	params SignUpUserParams,
) (SignUpUserResult, error) {
	if err := ensureEmailAvailable(ctx, ensureEmailAvailableParams{UserWriteRepository: d.UserWriteRepository, Email: params.Email}); err != nil {
		return SignUpUserResult{}, err
	}

	hashedPassword, err := password.Hash(params.Password)
	if err != nil {
		return SignUpUserResult{}, e.ErrPasswordHashingFailed{Cause: err}
	}

	user := models.New(models.NewUserParams{
		Email:        params.Email,
		PasswordHash: hashedPassword,
	})

	newUser, err := d.UserWriteRepository.SaveUser(ctx, user)
	if err != nil {
		return SignUpUserResult{}, err
	}

	userevents.DispatchUserSignedUp(ctx, d.EventDispatcher, userevents.NewUserSignedUpParams{
		UserID: newUser.ID,
		Email:  newUser.Email,
	})

	generateEmailVerificationChallenge(ctx, generateEmailVerificationChallengeParams{
		GenerateAuthChallenge: d.GenerateAuthChallenge,
		CreatedUser:           newUser,
	})

	return SignUpUserResult{User: newUser}, nil
}

type generateEmailVerificationChallengeParams struct {
	GenerateAuthChallenge GenerateAuthChallenge
	CreatedUser           models.User
}

func generateEmailVerificationChallenge(ctx context.Context, params generateEmailVerificationChallengeParams) {
	createdUser := params.CreatedUser
	if params.GenerateAuthChallenge == nil {
		slog.Error(e.ChallengeGeneratorMissingText, "userId", createdUser.ID)
		return
	}

	_, err := params.GenerateAuthChallenge(ctx, challenge.GenerateParams{
		UserID:     createdUser.ID,
		Target:     createdUser.Email,
		TargetType: challenge.TargetTypeEmail,
		Purpose:    challenge.PurposeEmailVerification,
	})
	if err != nil {
		slog.Error(e.GenerateAuthChallengeFailedText, "userId", createdUser.ID, "error", err)
	}
}

type ensureEmailAvailableParams struct {
	UserWriteRepository repositories.UserRepository
	Email               string
}

func ensureEmailAvailable(ctx context.Context, params ensureEmailAvailableParams) error {
	_, err := params.UserWriteRepository.FindUserByEmail(ctx, params.Email)
	if errors.As(err, &e.ErrUserNotFound{}) {
		return nil
	}
	if err != nil {
		return err
	}

	return e.ErrEmailTaken{Email: params.Email}
}
