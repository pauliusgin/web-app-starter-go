package main

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pauliusgin/web_app_starter-be/internal/scheduler"
)

// schedulerTestBudget bounds waits on a real cron tick, which the library rounds up to one second.
const schedulerTestBudget = 8 * time.Second

func newBufferedLogger() (*slog.Logger, *bytes.Buffer) {
	logOutput := &bytes.Buffer{}
	return slog.New(slog.NewTextHandler(logOutput, nil)), logOutput
}

func TestBuildSchedulerRegistersBackgroundJobs(t *testing.T) {
	logger, _ := newBufferedLogger()

	// The cleanup job only touches the database when it fires at midnight, never during the build.
	jobScheduler, err := buildScheduler(schedulerParams{Logger: logger})
	if err != nil {
		t.Fatalf("buildScheduler = %v, want nil", err)
	}

	stopScheduler(stopSchedulerParams{
		Scheduler: jobScheduler,
		Timeout:   time.Second,
		Logger:    logger,
	})
}

func TestNewSchedulerWithJobsReportsARejectedJob(t *testing.T) {
	logger, _ := newBufferedLogger()

	_, err := newSchedulerWithJobs(newSchedulerWithJobsParams{
		Logger: logger,
		Jobs: []scheduler.Job{
			{
				Name:     "timeout left unset",
				Schedule: scheduler.DailyAtMidnight,
				Run:      func(context.Context) error { return nil },
			},
		},
	})

	if err == nil {
		t.Fatal("newSchedulerWithJobs = nil error, want the job rejected")
	}
}

func TestStopSchedulerStopsAnIdleScheduler(t *testing.T) {
	logger, logOutput := newBufferedLogger()

	jobScheduler := scheduler.New(scheduler.Params{Logger: logger})
	jobScheduler.Start()

	stopScheduler(stopSchedulerParams{
		Scheduler: jobScheduler,
		Timeout:   time.Second,
		Logger:    logger,
	})

	if strings.Contains(logOutput.String(), "level=ERROR") {
		t.Errorf("stopScheduler logged an error, want a clean stop:\n%s", logOutput.String())
	}
}

func TestStopSchedulerLogsAStopThatTimesOut(t *testing.T) {
	logger, logOutput := newBufferedLogger()

	jobStarted := make(chan struct{})
	var signalJobStarted sync.Once
	releaseJob := make(chan struct{})
	t.Cleanup(func() { close(releaseJob) })

	jobScheduler := scheduler.New(scheduler.Params{Logger: logger})
	err := jobScheduler.RegisterJob(scheduler.Job{
		Name:     "blocks until released",
		Schedule: "@every 1s",
		Timeout:  schedulerTestBudget,
		Run: func(context.Context) error {
			signalJobStarted.Do(func() { close(jobStarted) })
			<-releaseJob
			return nil
		},
	})
	if err != nil {
		t.Fatalf("RegisterJob = %v, want nil", err)
	}

	jobScheduler.Start()

	select {
	case <-jobStarted:
	case <-time.After(schedulerTestBudget):
		t.Fatalf("job did not start within %s", schedulerTestBudget)
	}

	stopScheduler(stopSchedulerParams{
		Scheduler: jobScheduler,
		Timeout:   10 * time.Millisecond,
		Logger:    logger,
	})

	if !strings.Contains(logOutput.String(), "level=ERROR") {
		t.Errorf("stopScheduler logged no error, want the timed-out stop reported:\n%s",
			logOutput.String())
	}
}
