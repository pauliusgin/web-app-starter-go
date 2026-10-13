// Package scheduler runs recurring jobs on a cron schedule. It is a driving adapter: it calls
// into use cases on a timer the same way the HTTP layer calls into them on a request, and it
// knows nothing about what any individual job does.
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
)

// DailyAtMidnight runs a job once a day at 00:00 in the scheduler's location.
const DailyAtMidnight = "0 0 * * *"

// Job is one unit of recurring work handed to the scheduler.
type Job struct {
	Name     string
	Schedule string
	Timeout  time.Duration
	Run      func(context.Context) error
}

// Params carries the time zone schedules are read in and the logger job outcomes go to.
type Params struct {
	Location *time.Location
	Logger   *slog.Logger
}

// Scheduler owns a cron loop and the jobs registered on it.
type Scheduler struct {
	cron   *cron.Cron
	logger *slog.Logger
}

// New builds a scheduler that runs nothing until Start is called.
func New(params Params) Scheduler {
	location := params.Location
	if location == nil {
		location = time.UTC
	}

	logger := params.Logger
	if logger == nil {
		logger = slog.Default()
	}

	jobLogger := newCronLogger(logger)

	return Scheduler{
		cron: cron.New(
			cron.WithLocation(location),
			cron.WithLogger(jobLogger),
			cron.WithChain(cron.Recover(jobLogger)),
		),
		logger: logger,
	}
}

// RegisterJob validates the job and adds it to the cron loop, whether or not the loop is running.
func (s Scheduler) RegisterJob(job Job) error {
	if job.Run == nil {
		return ErrJobRunMissing{Name: job.Name}
	}

	if job.Timeout <= 0 {
		return ErrJobTimeoutInvalid{Name: job.Name, Timeout: job.Timeout}
	}

	_, err := s.cron.AddFunc(job.Schedule, func() {
		s.runJobOnce(job)
	})
	if err != nil {
		return ErrJobScheduleInvalid{Name: job.Name, Schedule: job.Schedule, Cause: err}
	}

	return nil
}

// Start begins running registered jobs on their schedules and returns immediately.
func (s Scheduler) Start() {
	s.cron.Start()
}

// Stop halts the schedule and waits for in-flight jobs until ctx runs out.
func (s Scheduler) Stop(ctx context.Context) error {
	jobsFinished := s.cron.Stop()

	select {
	case <-jobsFinished.Done():
		return nil
	case <-ctx.Done():
		return ErrSchedulerStopTimedOut{}
	}
}

func (s Scheduler) runJobOnce(job Job) {
	ctx, cancel := context.WithTimeout(context.Background(), job.Timeout)
	defer cancel()

	startedAt := time.Now()
	err := job.Run(ctx)
	elapsed := time.Since(startedAt)

	if err != nil {
		s.logger.Error("scheduled job failed",
			"job", job.Name, "duration", elapsed, "error", err)
		return
	}

	s.logger.Info("scheduled job finished", "job", job.Name, "duration", elapsed)
}
