package usecases

import (
	"context"

	"github.com/google/uuid"

	"github.com/pauliusgin/web_app_starter-be/internal/auth/challenge"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/token"
)

type findOwnedAuthChallengeParams struct {
	FindAuthChallengeByTokenHash FindAuthChallengeByTokenHash
	VerificationToken            string
	UserID                       uuid.UUID
	Purpose                      challenge.Purpose
}

func findOwnedAuthChallenge(
	ctx context.Context,
	params findOwnedAuthChallengeParams,
) (challenge.Challenge, error) {
	matched, err := params.FindAuthChallengeByTokenHash(ctx, token.Hash(params.VerificationToken))
	if err != nil {
		return challenge.Challenge{}, err
	}

	if matched.Purpose != params.Purpose {
		return challenge.Challenge{}, challenge.ErrChallengePurposeMismatch{
			Want: params.Purpose,
			Got:  matched.Purpose,
		}
	}
	if matched.UserID != params.UserID {
		return challenge.Challenge{}, challenge.ErrChallengeOwnedByAnotherUser{ChallengeID: matched.ID}
	}

	return matched, nil
}
