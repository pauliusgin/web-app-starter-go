package scheduler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testTick = "@every 1s"

const tickBudget = 8 * time.Second

func newTestScheduler(t *testing.T) Scheduler {
	t.Helper()

	return New(Params{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
}

func stopTestScheduler(t *testing.T, scheduler Scheduler) {
	t.Helper()

	stopCtx, cancelStop := context.WithTimeout(context.Background(), tickBudget)
	defer cancelStop()

	if err := scheduler.Stop(stopCtx); err != nil {
		t.Errorf("stopping the scheduler failed: %v", err)
	}
}

func startTestScheduler(t *testing.T, scheduler Scheduler) {
	t.Helper()

	scheduler.Start()

	t.Cleanup(func() { stopTestScheduler(t, scheduler) })
}

func awaitSignal(t *testing.T, signal <-chan struct{}, failure string) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(tickBudget):
		t.Fatal(failure)
	}
}

type blockingJob struct {
	started     chan struct{}
	startedOnce sync.Once
	release     chan struct{}
	releaseOnce sync.Once
	finished    atomic.Bool
}

func registerBlockingJob(t *testing.T, scheduler Scheduler) *blockingJob {
	t.Helper()

	job := &blockingJob{started: make(chan struct{}), release: make(chan struct{})}
	t.Cleanup(job.releaseRun)

	err := scheduler.RegisterJob(Job{
		Name:     "slow",
		Schedule: testTick,
		Timeout:  tickBudget,
		Run: func(context.Context) error {
			job.startedOnce.Do(func() { close(job.started) })
			<-job.release
			job.finished.Store(true)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("registering the job failed: %v", err)
	}

	return job
}

func (j *blockingJob) releaseRun() {
	j.releaseOnce.Do(func() { close(j.release) })
}

func TestNewSchedulerFallsBackToUTCAndTheDefaultLogger(t *testing.T) {
	scheduler := New(Params{})

	if scheduler.cron.Location() != time.UTC {
		t.Errorf("got location %v, want UTC", scheduler.cron.Location())
	}
	if scheduler.logger == nil {
		t.Error("got a nil logger, want the slog default")
	}
}

func TestRegisterJobAcceptsDailyAtMidnight(t *testing.T) {
	scheduler := newTestScheduler(t)

	err := scheduler.RegisterJob(Job{
		Name:     "nightly",
		Schedule: DailyAtMidnight,
		Timeout:  time.Minute,
		Run:      func(context.Context) error { return nil },
	})
	if err != nil {
		t.Fatalf("registering %q failed: %v", DailyAtMidnight, err)
	}
	if entries := scheduler.cron.Entries(); len(entries) != 1 {
		t.Errorf("got %d registered entries, want 1", len(entries))
	}
}

func TestRegisterJobRejectsAScheduleTheParserCannotRead(t *testing.T) {
	scheduler := newTestScheduler(t)

	err := scheduler.RegisterJob(Job{
		Name:     "nonsense",
		Schedule: "not a cron spec",
		Timeout:  time.Second,
		Run:      func(context.Context) error { return nil },
	})

	var scheduleErr ErrJobScheduleInvalid
	if !errors.As(err, &scheduleErr) {
		t.Fatalf("got %v, want ErrJobScheduleInvalid", err)
	}
	if scheduleErr.Name != "nonsense" || scheduleErr.Schedule != "not a cron spec" {
		t.Errorf("got %+v, want the rejected job's name and schedule", scheduleErr)
	}
	if scheduleErr.Cause == nil {
		t.Error("got a nil cause, want the parser's error wrapped")
	}
	if entries := scheduler.cron.Entries(); len(entries) != 0 {
		t.Errorf("got %d registered entries, want none", len(entries))
	}
}

func TestRegisterJobRejectsAJobWithNothingToRun(t *testing.T) {
	scheduler := newTestScheduler(t)

	err := scheduler.RegisterJob(Job{
		Name:     "empty",
		Schedule: testTick,
		Timeout:  time.Second,
	})

	var runMissingErr ErrJobRunMissing
	if !errors.As(err, &runMissingErr) {
		t.Fatalf("got %v, want ErrJobRunMissing", err)
	}
	if runMissingErr.Name != "empty" {
		t.Errorf("got name %q, want %q", runMissingErr.Name, "empty")
	}
	if entries := scheduler.cron.Entries(); len(entries) != 0 {
		t.Errorf("got %d registered entries, want none", len(entries))
	}
}

// A zero timeout is rejected rather than treated as "no deadline": context.WithTimeout(ctx, 0)
// produces an already-expired context, so accepting it would cancel every run before it started.
func TestRegisterJobRejectsATimeoutThatIsNotPositive(t *testing.T) {
	cases := []struct {
		name    string
		timeout time.Duration
	}{
		{name: "zero", timeout: 0},
		{name: "negative", timeout: -time.Second},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			scheduler := newTestScheduler(t)

			err := scheduler.RegisterJob(Job{
				Name:     "unbounded",
				Schedule: testTick,
				Timeout:  testCase.timeout,
				Run:      func(context.Context) error { return nil },
			})

			var timeoutErr ErrJobTimeoutInvalid
			if !errors.As(err, &timeoutErr) {
				t.Fatalf("got %v, want ErrJobTimeoutInvalid", err)
			}
			if timeoutErr.Timeout != testCase.timeout {
				t.Errorf("got timeout %s, want %s", timeoutErr.Timeout, testCase.timeout)
			}
			if entries := scheduler.cron.Entries(); len(entries) != 0 {
				t.Errorf("got %d registered entries, want none", len(entries))
			}
		})
	}
}

func TestRegisterJobRunsTheJobWithADeadlineOnItsContext(t *testing.T) {
	scheduler := newTestScheduler(t)

	const jobTimeout = 4 * time.Second
	deadlines := make(chan time.Time, 1)

	err := scheduler.RegisterJob(Job{
		Name:     "deadline-probe",
		Schedule: testTick,
		Timeout:  jobTimeout,
		Run: func(ctx context.Context) error {
			deadline, hasDeadline := ctx.Deadline()
			if !hasDeadline {
				deadline = time.Time{}
			}

			select {
			case deadlines <- deadline:
			default:
			}

			return nil
		},
	})
	if err != nil {
		t.Fatalf("registering the job failed: %v", err)
	}

	startedAt := time.Now()
	startTestScheduler(t, scheduler)

	select {
	case deadline := <-deadlines:
		if deadline.IsZero() {
			t.Fatal("the job's context carried no deadline")
		}

		earliest := startedAt.Add(jobTimeout)
		latest := time.Now().Add(jobTimeout)
		if deadline.Before(earliest) || deadline.After(latest) {
			t.Errorf("got deadline %s, want it between %s and %s", deadline, earliest, latest)
		}
	case <-time.After(tickBudget):
		t.Fatal("the job never ran")
	}
}

func TestAFailingJobIsLoggedWithItsNameAndError(t *testing.T) {
	logs := &bytes.Buffer{}
	scheduler := New(Params{Logger: slog.New(slog.NewTextHandler(logs, nil))})
	ran := make(chan struct{})
	ranOnce := sync.Once{}

	err := scheduler.RegisterJob(Job{
		Name:     "always-fails",
		Schedule: testTick,
		Timeout:  time.Second,
		Run: func(context.Context) error {
			ranOnce.Do(func() { close(ran) })
			return errors.New("the sweep failed")
		},
	})
	if err != nil {
		t.Fatalf("registering the job failed: %v", err)
	}

	scheduler.Start()
	awaitSignal(t, ran, "the job never ran")
	// Stop waits for the in-flight run, so its log line is written before the buffer is read.
	stopTestScheduler(t, scheduler)

	for _, want := range []string{"level=ERROR", "scheduled job failed", "job=always-fails", "the sweep failed"} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("logs = %s, want them to carry %q", logs, want)
		}
	}
}

// Without the cron.Recover chain a panic on a job goroutine crashes the whole test binary, so
// surviving one run proves the panic was contained.
func TestAPanickingJobDoesNotCrashTheProcess(t *testing.T) {
	scheduler := newTestScheduler(t)
	ran := make(chan struct{})
	ranOnce := sync.Once{}

	err := scheduler.RegisterJob(Job{
		Name:     "always-panics",
		Schedule: testTick,
		Timeout:  time.Second,
		Run: func(context.Context) error {
			ranOnce.Do(func() { close(ran) })
			panic("the job exploded")
		},
	})
	if err != nil {
		t.Fatalf("registering the job failed: %v", err)
	}

	scheduler.Start()
	awaitSignal(t, ran, "the job never ran")
	stopTestScheduler(t, scheduler)
}

func TestStopWaitsForAJobThatIsStillRunning(t *testing.T) {
	scheduler := newTestScheduler(t)
	job := registerBlockingJob(t, scheduler)

	scheduler.Start()
	awaitSignal(t, job.started, "the job never started")

	stopResults := make(chan error, 1)
	go func() {
		stopCtx, cancelStop := context.WithTimeout(context.Background(), tickBudget)
		defer cancelStop()

		stopResults <- scheduler.Stop(stopCtx)
	}()

	select {
	case stopErr := <-stopResults:
		t.Fatalf("Stop returned %v while the job was still running", stopErr)
	case <-time.After(200 * time.Millisecond):
	}

	job.releaseRun()

	select {
	case stopErr := <-stopResults:
		if stopErr != nil {
			t.Fatalf("got %v, want nil", stopErr)
		}
	case <-time.After(tickBudget):
		t.Fatal("Stop never returned after the job was released")
	}

	if !job.finished.Load() {
		t.Error("Stop returned before the job finished")
	}
}

func TestStopReportsATimeoutWhenTheGracePeriodIsAlreadySpent(t *testing.T) {
	scheduler := newTestScheduler(t)
	job := registerBlockingJob(t, scheduler)

	scheduler.Start()
	awaitSignal(t, job.started, "the job never started")

	spentCtx, cancelSpent := context.WithCancel(context.Background())
	cancelSpent()

	var stopTimedOutErr ErrSchedulerStopTimedOut
	if stopErr := scheduler.Stop(spentCtx); !errors.As(stopErr, &stopTimedOutErr) {
		t.Fatalf("got %v, want ErrSchedulerStopTimedOut", stopErr)
	}

	job.releaseRun()
	stopTestScheduler(t, scheduler)
}
