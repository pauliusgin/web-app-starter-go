package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/brokers/rabbitmq"
	auth "github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/auth/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	uploadsrepo "github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/uploads/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/auth/refreshtoken"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/gateways/storage"
	uploadusecases "github.com/pauliusgin/web_app_starter-be/internal/domains/uploads/usecases"
	"github.com/pauliusgin/web_app_starter-be/internal/scheduler"
)

const refreshTokenCleanupJobName = "refresh_token_cleanup"
const refreshTokenCleanupMessage = "unused refresh tokens purged"

const refreshTokenCleanupTimeout = 5 * time.Minute

const pendingUploadCleanupJobName = "pending_upload_cleanup"
const pendingUploadCleanupMessage = "stale pending uploads purged"

const pendingUploadCleanupTimeout = 10 * time.Minute

type schedulerParams struct {
	Database       *sql.DB
	Logger         *slog.Logger
	EventPublisher rabbitmq.EventPublisher
	FileStorage    storage.FileStorage
}

func buildScheduler(params schedulerParams) (scheduler.Scheduler, error) {
	logger := params.Logger
	if logger == nil {
		logger = slog.Default()
	}

	params.Logger = logger

	return newSchedulerWithJobs(newSchedulerWithJobsParams{
		Logger: logger,
		Jobs: []scheduler.Job{
			refreshTokenCleanupJob(params),
			pendingUploadCleanupJob(params),
		},
	})
}

func refreshTokenCleanupJob(params schedulerParams) scheduler.Job {
	cleanupDependencies := refreshtoken.UnusedDeletionDependencies{
		WriteRepository: auth.NewRefreshTokenWriteRepository(sqlc.New(params.Database)),
	}

	deleteRefreshTokens := func(ctx context.Context) error {
		deletedCount, err := refreshtoken.DeleteUnusedRefreshTokens(ctx, cleanupDependencies,
			refreshtoken.UnusedDeletionParams{CleanUpThreshold: refreshtoken.CleanUpThresholdInDays})
		if err != nil {
			return err
		}

		params.Logger.Info(refreshTokenCleanupMessage,
			"job", refreshTokenCleanupJobName, "deleted", deletedCount)
		return nil
	}

	return scheduler.Job{
		Name:     refreshTokenCleanupJobName,
		Schedule: scheduler.DailyAtMidnight,
		Timeout:  refreshTokenCleanupTimeout,
		Run:      deleteRefreshTokens,
	}
}

func pendingUploadCleanupJob(params schedulerParams) scheduler.Job {
	queries := sqlc.New(params.Database)
	cleanupDependencies := uploadusecases.DeleteStalePendingUploadsDependencies{
		UploadRepository: uploadsrepo.NewUploadRepository(queries),
		FileStorage:      params.FileStorage,
		EventDispatcher: buildEventDispatcher(buildEventDispatcherParams{
			Queries:   queries,
			Publisher: params.EventPublisher.PublishEvent,
		}),
	}

	deleteStalePendingUploads := func(ctx context.Context) error {
		result, err := uploadusecases.DeleteStalePendingUploads(ctx, cleanupDependencies)

		params.Logger.Info(pendingUploadCleanupMessage,
			"job", pendingUploadCleanupJobName, "deleted", result.DeletedCount)

		return err
	}

	return scheduler.Job{
		Name:     pendingUploadCleanupJobName,
		Schedule: scheduler.DailyAtMidnight,
		Timeout:  pendingUploadCleanupTimeout,
		Run:      deleteStalePendingUploads,
	}
}

type newSchedulerWithJobsParams struct {
	Logger *slog.Logger
	Jobs   []scheduler.Job
}

func newSchedulerWithJobs(params newSchedulerWithJobsParams) (scheduler.Scheduler, error) {
	jobScheduler := scheduler.New(scheduler.Params{
		Location: time.UTC,
		Logger:   params.Logger,
	})

	cronJobList := make([]string, 0, len(params.Jobs))

	for _, job := range params.Jobs {
		if err := jobScheduler.RegisterJob(job); err != nil {
			return scheduler.Scheduler{}, err
		}

		cronJobItem := fmt.Sprintf("\x1b[34m [ cronjob ready: %s ] \x1b[0m\n", job.Name)
		cronJobList = append(cronJobList, cronJobItem)

		params.Logger.Info("background job registered",
			"job", job.Name, "schedule", job.Schedule, "timeout", job.Timeout)
	}

	fmt.Println("")
	for _, cronJobItem := range cronJobList {
		fmt.Print(cronJobItem)
	}
	fmt.Println("")

	return jobScheduler, nil
}

type stopSchedulerParams struct {
	Scheduler scheduler.Scheduler
	Timeout   time.Duration
	Logger    *slog.Logger
}

func stopScheduler(params stopSchedulerParams) {
	logger := params.Logger
	if logger == nil {
		logger = slog.Default()
	}

	stopCtx, cancelStop := context.WithTimeout(context.Background(), params.Timeout)
	defer cancelStop()

	if err := params.Scheduler.Stop(stopCtx); err != nil {
		logger.Error("scheduler did not stop cleanly", "error", err)
		return
	}

	logger.Info("scheduler stopped")
}
