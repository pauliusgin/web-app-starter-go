package challenge

import (
	"fmt"

	"github.com/google/uuid"
)

const targetMissingText = "expected the address or number the challenge is sent to"
const targetTypeMissingText = "expected the kind of target the challenge is sent to"
const purposeMissingText = "expected what completing the challenge proves"
const saveChallengeFailedText = "failed saving the challenge"
const challengeNotFoundText = "no live challenge matches the presented token"
const challengeLookupFailedText = "failed finding the challenge by its token hash"
const consumeChallengeFailedText = "failed spending the challenge"
const challengePurposeMismatchText = "the challenge was minted to prove something else"
const challengeOwnedByAnotherUserText = "the challenge was minted for a different account"

// ErrTargetMissing reports a challenge asked for without the target it would be sent to.
type ErrTargetMissing struct{}

func (e ErrTargetMissing) Error() string {
	return fmt.Sprintf("missing target: %s", targetMissingText)
}

// ErrTargetTypeMissing reports a challenge asked for without a target type.
type ErrTargetTypeMissing struct{}

func (e ErrTargetTypeMissing) Error() string {
	return fmt.Sprintf("missing target type: %s", targetTypeMissingText)
}

// ErrPurposeMissing reports a challenge asked for without a purpose.
type ErrPurposeMissing struct{}

func (e ErrPurposeMissing) Error() string {
	return fmt.Sprintf("missing purpose: %s", purposeMissingText)
}

// ErrSaveChallengeFailed reports a challenge the database refused to store.
type ErrSaveChallengeFailed struct {
	Cause error
}

func (e ErrSaveChallengeFailed) Error() string {
	return fmt.Sprintf("%s: %s", saveChallengeFailedText, e.Cause)
}

func (e ErrSaveChallengeFailed) Unwrap() error {
	return e.Cause
}

// ErrChallengeNotFound reports a presented token that matches no live stored challenge.
type ErrChallengeNotFound struct{}

func (e ErrChallengeNotFound) Error() string {
	return fmt.Sprintf("unknown challenge: %s", challengeNotFoundText)
}

// ErrChallengeLookupFailed reports a lookup by token hash the database could not complete.
type ErrChallengeLookupFailed struct {
	Cause error
}

func (e ErrChallengeLookupFailed) Error() string {
	return fmt.Sprintf("%s: %s", challengeLookupFailedText, e.Cause)
}

func (e ErrChallengeLookupFailed) Unwrap() error {
	return e.Cause
}

// ErrConsumeChallengeFailed reports a used_at stamp the database refused to write.
type ErrConsumeChallengeFailed struct {
	Cause error
}

func (e ErrConsumeChallengeFailed) Error() string {
	return fmt.Sprintf("%s: %s", consumeChallengeFailedText, e.Cause)
}

func (e ErrConsumeChallengeFailed) Unwrap() error {
	return e.Cause
}

// ErrChallengePurposeMismatch reports a challenge presented for a purpose it was not minted for.
type ErrChallengePurposeMismatch struct {
	Want Purpose
	Got  Purpose
}

func (e ErrChallengePurposeMismatch) Error() string {
	return fmt.Sprintf("challenge purpose %q, want %q: %s", e.Got, e.Want, challengePurposeMismatchText)
}

// ErrChallengeOwnedByAnotherUser reports a challenge redeemed by an account it was not minted for.
type ErrChallengeOwnedByAnotherUser struct {
	ChallengeID uuid.UUID
}

func (e ErrChallengeOwnedByAnotherUser) Error() string {
	return fmt.Sprintf("challenge %s belongs to another user: %s",
		e.ChallengeID, challengeOwnedByAnotherUserText)
}
