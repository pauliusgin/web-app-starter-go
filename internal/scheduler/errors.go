package scheduler

import (
	"fmt"
	"time"
)

const jobScheduleInvalidText = "expected a cron spec the standard parser accepts"

const jobRunMissingText = "expected a function to run"

const jobTimeoutInvalidText = "expected a positive per-run timeout"

const schedulerStopTimedOutText = "a job was still running when the shutdown grace period ran out"

// ErrJobScheduleInvalid reports a job whose cron spec the parser refused.
type ErrJobScheduleInvalid struct {
	Name     string
	Schedule string
	Cause    error
}

func (e ErrJobScheduleInvalid) Error() string {
	return fmt.Sprintf("invalid schedule %q for job %q: %s: %s",
		e.Schedule, e.Name, jobScheduleInvalidText, e.Cause)
}

func (e ErrJobScheduleInvalid) Unwrap() error {
	return e.Cause
}

// ErrJobRunMissing reports a job registered with nothing to run.
type ErrJobRunMissing struct {
	Name string
}

func (e ErrJobRunMissing) Error() string {
	return fmt.Sprintf("no work for job %q: %s", e.Name, jobRunMissingText)
}

// ErrJobTimeoutInvalid reports a job registered with a zero or negative per-run timeout.
type ErrJobTimeoutInvalid struct {
	Name    string
	Timeout time.Duration
}

func (e ErrJobTimeoutInvalid) Error() string {
	return fmt.Sprintf("invalid timeout %s for job %q: %s",
		e.Timeout, e.Name, jobTimeoutInvalidText)
}

// ErrSchedulerStopTimedOut reports a shutdown that gave up before its jobs finished.
type ErrSchedulerStopTimedOut struct{}

func (e ErrSchedulerStopTimedOut) Error() string {
	return schedulerStopTimedOutText
}
