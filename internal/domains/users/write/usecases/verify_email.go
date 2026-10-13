package usecases

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	e "github.com/pauliusgin/web_app_starter-be/internal/domains/users/errors"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/models"
	writerepo "github.com/pauliusgin/web_app_starter-be/internal/domains/users/write/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
	userevents "github.com/pauliusgin/web_app_starter-be/internal/events/user"
)

// VerifyEmailParams is the link the user clicked and the identity their access token proved.
type VerifyEmailParams struct {
	VerificationToken string
	UserID            uuid.UUID
}

// FindAuthChallengeByTokenHash is the challenge lookup call verifying an email depends on.
type FindAuthChallengeByTokenHash func(
	ctx context.Context,
	tokenHash string,
) (challenge.Challenge, error)

// ConsumeAuthChallenge is the challenge-spending call verifying an email depends on.
type ConsumeAuthChallenge func(ctx context.Context, challengeID uuid.UUID) error

// VerifyEmailDependencies carries the collaborators verifying an email needs.
type VerifyEmailDependencies struct {
	UserWriteRepository          writerepo.UserRepository
	EventDispatcher              events.Dispatcher
	FindAuthChallengeByTokenHash FindAuthChallengeByTokenHash
	ConsumeAuthChallenge         ConsumeAuthChallenge
}

// VerifyEmail spends the challenge the link carries and acts on the address it proved: the user's
// current email is marked verified, any other address becomes the user's new email.
func VerifyEmail(
	ctx context.Context,
	d VerifyEmailDependencies,
	params VerifyEmailParams,
) error {
	matched, err := findOwnedAuthChallenge(ctx, findOwnedAuthChallengeParams{
		FindAuthChallengeByTokenHash: d.FindAuthChallengeByTokenHash,
		VerificationToken:            params.VerificationToken,
		UserID:                       params.UserID,
		Purpose:                      challenge.PurposeEmailVerification,
	})
	if errors.As(err, &challenge.ErrChallengeNotFound{}) {
		return answerForAMissingChallenge(ctx, d, params.UserID)
	}
	if err != nil {
		return err
	}

	storedUser, err := d.UserWriteRepository.FindUserByID(ctx, params.UserID)
	if err != nil {
		return err
	}

	if matched.Target == storedUser.Email {
		return markCurrentEmailVerified(ctx, d, matched)
	}

	return switchToProvenEmail(ctx, d, switchToProvenEmailParams{
		Challenge:     matched,
		PreviousEmail: storedUser.Email,
	})
}

func markCurrentEmailVerified(
	ctx context.Context,
	d VerifyEmailDependencies,
	matched challenge.Challenge,
) error {
	verified, err := d.UserWriteRepository.MarkEmailVerified(ctx, matched.UserID)
	if err != nil {
		return err
	}
	if err := d.ConsumeAuthChallenge(ctx, matched.ID); err != nil {
		return err
	}

	userevents.DispatchEmailVerified(ctx, d.EventDispatcher, userevents.EmailVerifiedParams{
		UserID: verified.ID,
		Email:  verified.Email,
	})

	return nil
}

type switchToProvenEmailParams struct {
	Challenge     challenge.Challenge
	PreviousEmail string
}

func switchToProvenEmail(
	ctx context.Context,
	d VerifyEmailDependencies,
	params switchToProvenEmailParams,
) error {
	changedUser, err := d.UserWriteRepository.ChangeEmail(ctx, models.ChangeEmailParams{
		ID:    params.Challenge.UserID,
		Email: params.Challenge.Target,
	})
	if err != nil {
		return err
	}
	if err := d.ConsumeAuthChallenge(ctx, params.Challenge.ID); err != nil {
		return err
	}

	userevents.DispatchUserEmailChanged(ctx, d.EventDispatcher, userevents.NewUserEmailChangedParams{
		UserID:        changedUser.ID,
		PreviousEmail: params.PreviousEmail,
		Email:         changedUser.Email,
	})

	return nil
}

func answerForAMissingChallenge(
	ctx context.Context,
	d VerifyEmailDependencies,
	userID uuid.UUID,
) error {
	storedUser, err := d.UserWriteRepository.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}

	emailVerified := !storedUser.EmailVerifiedAt.IsZero()

	if emailVerified {
		return nil
	}

	return e.ErrVerificationLinkInvalid{}
}
