package challenge

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// IssuanceMinimumGap is the least time between two challenges a user sends to the same target.
const IssuanceMinimumGap = 30 * time.Second

// IssuanceWindow is the rolling period the per-target and per-user issuance limits count over.
const IssuanceWindow = 24 * time.Hour

// IssuanceLimitPerTarget caps the challenges a user sends to one target within IssuanceWindow.
const IssuanceLimitPerTarget = 5

// IssuanceLimitPerUser caps the challenges one user is minted within IssuanceWindow, across targets.
const IssuanceLimitPerUser = 10

// EnsureIssuanceAllowedDependencies carries the challenge history and the clock the limits are judged by.
type EnsureIssuanceAllowedDependencies struct {
	ReadRepository ReadRepository
	Now            func() time.Time
}

// EnsureIssuanceAllowedParams is the challenge about to be asked for.
type EnsureIssuanceAllowedParams struct {
	UserID  uuid.UUID
	Target  string
	Purpose Purpose
}

// EnsureIssuanceAllowed returns ErrIssuanceRateLimited when another challenge would break a limit.
func EnsureIssuanceAllowed(
	ctx context.Context,
	d EnsureIssuanceAllowedDependencies,
	params EnsureIssuanceAllowedParams,
) error {
	now := d.Now()
	createdAfter := now.Add(-IssuanceWindow)

	targetHistory, err := d.ReadRepository.ListCreatedAtByTarget(ctx, ListCreatedAtByTargetParams{
		UserID:       params.UserID,
		Target:       params.Target,
		Purpose:      params.Purpose,
		CreatedAfter: createdAfter,
	})
	if err != nil {
		return err
	}

	userHistory, err := d.ReadRepository.ListCreatedAtByUser(ctx, ListCreatedAtByUserParams{
		UserID:       params.UserID,
		Purpose:      params.Purpose,
		CreatedAfter: createdAfter,
	})
	if err != nil {
		return err
	}

	retryAfter := max(
		waitForMinimumGap(waitForMinimumGapParams{History: targetHistory, Now: now}),
		waitForWindowSlot(waitForWindowSlotParams{History: targetHistory, Limit: IssuanceLimitPerTarget, Now: now}),
		waitForWindowSlot(waitForWindowSlotParams{History: userHistory, Limit: IssuanceLimitPerUser, Now: now}),
	)
	if retryAfter > 0 {
		return ErrIssuanceRateLimited{RetryAfter: retryAfter}
	}

	return nil
}

type waitForMinimumGapParams struct {
	History []time.Time
	Now     time.Time
}

func waitForMinimumGap(params waitForMinimumGapParams) time.Duration {
	if len(params.History) == 0 {
		return 0
	}

	latest := params.History[len(params.History)-1]

	return latest.Add(IssuanceMinimumGap).Sub(params.Now)
}

type waitForWindowSlotParams struct {
	History []time.Time
	Limit   int
	Now     time.Time
}

// waitForWindowSlot relies on History being oldest first: the window frees a slot once the
// Limit-th newest entry ages out of it.
func waitForWindowSlot(params waitForWindowSlotParams) time.Duration {
	if len(params.History) < params.Limit {
		return 0
	}

	oldestToAgeOut := params.History[len(params.History)-params.Limit]

	return oldestToAgeOut.Add(IssuanceWindow).Sub(params.Now)
}
