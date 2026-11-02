package handlers

import (
	"context"
	"time"
)

// ResponseClock is the time source a response floor is measured and waited out with.
type ResponseClock struct {
	Now  func() time.Time
	Wait func(ctx context.Context, duration time.Duration)
}

// SystemResponseClock reads the wall clock and waits on a timer.
var SystemResponseClock = ResponseClock{Now: time.Now, Wait: WaitOrCancel}

// WaitForResponseFloorParams is when the request arrived, the least time it may take and the clock to measure by.
type WaitForResponseFloorParams struct {
	StartedAt time.Time
	Floor     time.Duration
	Clock     ResponseClock
}

// WaitForResponseFloor blocks until Floor has passed since StartedAt, so the answer's timing says nothing
// about which path the request took.
func WaitForResponseFloor(ctx context.Context, params WaitForResponseFloorParams) {
	remaining := params.StartedAt.Add(params.Floor).Sub(params.Clock.Now())
	if remaining <= 0 {
		return
	}

	params.Clock.Wait(ctx, remaining)
}

// WaitOrCancel waits out the duration, returning early once ctx ends.
func WaitOrCancel(ctx context.Context, duration time.Duration) {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-timer.C:
	case <-ctx.Done():
	}
}
