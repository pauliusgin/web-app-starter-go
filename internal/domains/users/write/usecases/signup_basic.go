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
	writerepo "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	userevents "github.com/pauliusgin/web_app_starter-be/internal/events/user"
)

// SignupUserParams is the plaintext signup input; the password is hashed before storage.
type SignupUserParams struct {
	Email    string
	Password string
}

// SignupUserResult is the user the signup stored.
type SignupUserResult struct {
	User models.User
}

// GenerateAuthChallenge is the challenge call creating a user depends on.
type GenerateAuthChallenge func(
	ctx context.Context,
	params challenge.GenerateParams,
) (challenge.Generated, error)

// SignupUserDependencies carries the collaborators creating a user needs.
type SignupUserDependencies struct {
	UserWriteRepository   writerepo.UserRepository
	EventDispatcher       events.Dispatcher
	GenerateAuthChallenge GenerateAuthChallenge
}

// SignupUser hashes the password, stores the user, announces the signup and asks for verification.
func SignupUser(
	ctx context.Context,
	d SignupUserDependencies,
	params SignupUserParams,
) (SignupUserResult, error) {
	if err := ensureEmailAvailable(ctx, d.UserWriteRepository, params.Email); err != nil {
		return SignupUserResult{}, err
	}

	hashedPassword, err := password.Hash(params.Password)
	if err != nil {
		return SignupUserResult{}, e.ErrPasswordHashingFailed{Cause: err}
	}

	user := models.New(models.NewUserParams{
		Email:        params.Email,
		PasswordHash: hashedPassword,
	})

	newUser, err := d.UserWriteRepository.Save(ctx, user)
	if err != nil {
		return SignupUserResult{}, err
	}

	userevents.DispatchUserSignedUp(ctx, d.EventDispatcher, userevents.NewUserSignedUpParams{
		UserID: newUser.ID,
		Email:  newUser.Email,
	})

	generateEmailVerificationChallenge(ctx, d.GenerateAuthChallenge, newUser)

	return SignupUserResult{User: newUser}, nil
}

func generateEmailVerificationChallenge(ctx context.Context, generate GenerateAuthChallenge, createdUser models.User) {
	if generate == nil {
		slog.Error(e.ChallengeGeneratorMissingText, "userId", createdUser.ID)
		return
	}

	_, err := generate(ctx, challenge.GenerateParams{
		UserID:     createdUser.ID,
		Target:     createdUser.Email,
		TargetType: challenge.TargetTypeEmail,
		Purpose:    challenge.PurposeEmailVerification,
	})
	if err != nil {
		slog.Error(e.GenerateAuthChallengeFailedText, "userId", createdUser.ID, "error", err)
	}
}

func ensureEmailAvailable(ctx context.Context, userWriteRepository writerepo.UserRepository, email string) error {
	_, err := userWriteRepository.FindUserByEmail(ctx, email)
	if errors.As(err, &e.ErrUserNotFound{}) {
		return nil
	}
	if err != nil {
		return err
	}

	return e.ErrEmailTaken{Email: email}
}
