package handlers

import (
	"context"
	"testing"
	"time"
)

var floorStartedAt = time.Date(2026, time.November, 2, 9, 0, 0, 0, time.UTC)

// recordingClock reads a fixed now and records each wait instead of sleeping.
type recordingClock struct {
	now    time.Time
	waited []time.Duration
}

func (c *recordingClock) clock() ResponseClock {
	return ResponseClock{
		Now:  func() time.Time { return c.now },
		Wait: func(_ context.Context, duration time.Duration) { c.waited = append(c.waited, duration) },
	}
}

func TestWaitForResponseFloorWaitsOutOnlyTheRemainingTime(t *testing.T) {
	cases := []struct {
		name       string
		elapsed    time.Duration
		wantWaited []time.Duration
	}{
		{name: "fast path pads to the floor", elapsed: 150 * time.Millisecond, wantWaited: []time.Duration{850 * time.Millisecond}},
		{name: "at the floor", elapsed: time.Second},
		{name: "past the floor", elapsed: 3 * time.Second},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			clock := &recordingClock{now: floorStartedAt.Add(testCase.elapsed)}

			WaitForResponseFloor(context.Background(), WaitForResponseFloorParams{
				StartedAt: floorStartedAt,
				Floor:     time.Second,
				Clock:     clock.clock(),
			})

			if len(clock.waited) != len(testCase.wantWaited) {
				t.Fatalf("waited = %v, want %v", clock.waited, testCase.wantWaited)
			}
			for index, want := range testCase.wantWaited {
				if clock.waited[index] != want {
					t.Errorf("waited[%d] = %s, want %s", index, clock.waited[index], want)
				}
			}
		})
	}
}

func TestWaitOrCancelReturnsOnceTheContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	returned := make(chan struct{})
	go func() {
		WaitOrCancel(ctx, time.Hour)
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("WaitOrCancel kept waiting after the context ended")
	}
}

func TestWaitOrCancelWaitsOutTheDuration(t *testing.T) {
	startedAt := time.Now()

	WaitOrCancel(context.Background(), 20*time.Millisecond)

	if elapsed := time.Since(startedAt); elapsed < 20*time.Millisecond {
		t.Errorf("returned after %s, want at least 20ms", elapsed)
	}
}
