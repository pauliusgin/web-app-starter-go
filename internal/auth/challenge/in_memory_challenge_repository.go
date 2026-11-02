package challenge

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"
)

var _ WriteRepository = (*InMemoryChallengeRepository)(nil)
var _ ReadRepository = (*InMemoryChallengeRepository)(nil)

// InMemoryChallengeRepository is both challenge ports over one slice, answering as the auth_challenges
// queries do; Now stands in for the database clock and reads the wall clock when nil.
type InMemoryChallengeRepository struct {
	Challenges []Challenge
	Now        func() time.Time

	// SaveFailure, MarkUsedFailure, FindFailure and ListFailure, when set, are returned instead of doing the work.
	SaveFailure     error
	MarkUsedFailure error
	FindFailure     error
	ListFailure     error
}

// NewInMemoryChallengeRepository builds a repository holding the given challenges and no failure staged.
func NewInMemoryChallengeRepository(challenges ...Challenge) *InMemoryChallengeRepository {
	return &InMemoryChallengeRepository{Challenges: challenges}
}

// SaveChallenge records the challenge stamped with the repository clock, or returns SaveFailure when staged.
func (r *InMemoryChallengeRepository) SaveChallenge(_ context.Context, newChallenge NewChallenge) (Challenge, error) {
	if r.SaveFailure != nil {
		return Challenge{}, r.SaveFailure
	}

	saved := Challenge{
		ID:          newChallenge.ID,
		UserID:      newChallenge.UserID,
		TargetType:  newChallenge.TargetType,
		Target:      newChallenge.Target,
		Purpose:     newChallenge.Purpose,
		TokenHash:   newChallenge.TokenHash,
		Code:        newChallenge.Code,
		Provider:    newChallenge.Provider,
		ReferenceID: newChallenge.ReferenceID,
		CreatedAt:   r.now(),
		ExpiresAt:   newChallenge.ExpiresAt,
	}
	r.Challenges = append(r.Challenges, saved)

	return saved, nil
}

// MarkUsed stamps a live challenge used; a used, expired or unknown one is left as it is, without an error.
func (r *InMemoryChallengeRepository) MarkUsed(_ context.Context, challengeID uuid.UUID) error {
	if r.MarkUsedFailure != nil {
		return r.MarkUsedFailure
	}

	now := r.now()
	index := slices.IndexFunc(r.Challenges, func(stored Challenge) bool {
		return stored.ID == challengeID && !stored.IsUsed() && !stored.IsExpired(now)
	})
	if index >= 0 {
		r.Challenges[index].UsedAt = now
	}

	return nil
}

// FindByTokenHash returns the unused, unexpired challenge with the hash, or ErrChallengeNotFound.
func (r *InMemoryChallengeRepository) FindByTokenHash(_ context.Context, tokenHash string) (Challenge, error) {
	if r.FindFailure != nil {
		return Challenge{}, r.FindFailure
	}

	now := r.now()
	for _, stored := range r.Challenges {
		if stored.TokenHash == tokenHash && !stored.IsUsed() && !stored.IsExpired(now) {
			return stored, nil
		}
	}

	return Challenge{}, ErrChallengeNotFound{}
}

// ListCreatedAtByTarget returns, oldest first, when challenges for the purpose were sent to the target.
func (r *InMemoryChallengeRepository) ListCreatedAtByTarget(
	_ context.Context,
	params ListCreatedAtByTargetParams,
) ([]time.Time, error) {
	return r.listCreatedAt(listCreatedAtParams{
		Purpose:      params.Purpose,
		CreatedAfter: params.CreatedAfter,
		Matches: func(stored Challenge) bool {
			return stored.UserID == params.UserID && stored.Target == params.Target
		},
	})
}

// ListCreatedAtByUser returns, oldest first, when challenges for the purpose were minted for the user.
func (r *InMemoryChallengeRepository) ListCreatedAtByUser(
	_ context.Context,
	params ListCreatedAtByUserParams,
) ([]time.Time, error) {
	return r.listCreatedAt(listCreatedAtParams{
		Purpose:      params.Purpose,
		CreatedAfter: params.CreatedAfter,
		Matches:      func(stored Challenge) bool { return stored.UserID == params.UserID },
	})
}

type listCreatedAtParams struct {
	Purpose      Purpose
	CreatedAfter time.Time
	Matches      func(stored Challenge) bool
}

func (r *InMemoryChallengeRepository) listCreatedAt(params listCreatedAtParams) ([]time.Time, error) {
	if r.ListFailure != nil {
		return nil, r.ListFailure
	}

	var createdAt []time.Time
	for _, stored := range r.Challenges {
		if stored.Purpose == params.Purpose && stored.CreatedAt.After(params.CreatedAfter) && params.Matches(stored) {
			createdAt = append(createdAt, stored.CreatedAt)
		}
	}
	slices.SortFunc(createdAt, time.Time.Compare)

	return createdAt, nil
}

func (r *InMemoryChallengeRepository) now() time.Time {
	if r.Now == nil {
		return time.Now()
	}

	return r.Now()
}
