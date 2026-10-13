package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/brokers/rabbitmq"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres"
	messagingrepo "github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/messaging/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres/sqlc"
	"github.com/pauliusgin/web_app_starter-be/internal/api/http/routes"
	httpserver "github.com/pauliusgin/web_app_starter-be/internal/api/http/server"
	"github.com/pauliusgin/web_app_starter-be/internal/config"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/gateways/email"
	"github.com/pauliusgin/web_app_starter-be/internal/scheduler"
)

type startupSteps struct {
	LoadConfig              func() (config.Config, error)
	ConnectDatabase         func(ctx context.Context, databaseURL string) (*sql.DB, error)
	OpenEventBroker         func(params openEventBrokerParams) (eventBroker, error)
	BuildEmailGateway       func(brevoConfig config.BrevoConfig) (email.Gateway, error)
	BuildFileStorage        func(firebaseConfig config.FirebaseConfig) (fileStorageResource, error)
	BuildImageProcessor     func() (imageProcessorResource, error)
	StartEventConsumers     func(params startEventConsumersParams) (func(), error)
	BuildScheduler          func(params schedulerParams) (scheduler.Scheduler, error)
	BuildRouterDependencies func(params buildRouterDependenciesParams) (routes.RouterDependencies, error)
	ListenTCP               func(network string, address string) (net.Listener, error)
}

func serverStartupSteps() startupSteps {
	return startupSteps{
		LoadConfig:              config.Load,
		ConnectDatabase:         postgres.NewDatabaseConnection,
		OpenEventBroker:         openEventBroker,
		BuildEmailGateway:       buildEmailGateway,
		BuildFileStorage:        buildFileStorage,
		BuildImageProcessor:     buildImageProcessor,
		StartEventConsumers:     startEventConsumers,
		BuildScheduler:          buildScheduler,
		BuildRouterDependencies: buildRouterDependencies,
		ListenTCP:               net.Listen,
	}
}

func runServer(s startupSteps) error {
	cfg, err := s.LoadConfig()
	if err != nil {
		return err
	}

	emailGateway, err := s.BuildEmailGateway(cfg.Brevo)
	if err != nil {
		return err
	}

	fileStorage, err := s.BuildFileStorage(cfg.Firebase)
	if err != nil {
		return err
	}
	defer fileStorage.Close()

	imageProcessor, err := s.BuildImageProcessor()
	if err != nil {
		return err
	}
	defer imageProcessor.Close()

	db, err := s.ConnectDatabase(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	broker, err := s.OpenEventBroker(openEventBrokerParams{
		AmqpURL: cfg.RabbitMQURL,
		Logger:  slog.Default(),
	})
	if err != nil {
		return err
	}

	defer broker.Close()

	fmt.Println("\n\x1b[34m [ RabbitMQ server connection ready ] \x1b[0m")
	fmt.Println("")
	slog.Info("rabbitmq ready", "address", rabbitmq.SanitizeBrokerAddress(cfg.RabbitMQURL))

	queries := sqlc.New(db)

	stopEventConsumers, err := s.StartEventConsumers(startEventConsumersParams{
		Connection:        broker.Connection,
		AmqpURL:           cfg.RabbitMQURL,
		Logger:            slog.Default(),
		FrontendBaseURL:   cfg.FrontendBaseURL,
		EmailGateway:      emailGateway,
		MessageRepository: messagingrepo.NewMessageWriteRepository(queries),
		EventDispatcher: buildEventDispatcher(buildEventDispatcherParams{
			Queries:   queries,
			Publisher: broker.Publisher.PublishEvent,
		}),
	})
	if err != nil {
		return err
	}

	defer stopEventConsumers()

	jobScheduler, err := s.BuildScheduler(schedulerParams{
		Database:       db,
		Logger:         slog.Default(),
		EventPublisher: broker.Publisher,
		FileStorage:    fileStorage.FileStorage,
	})
	if err != nil {
		return err
	}

	jobScheduler.Start()

	defer stopScheduler(stopSchedulerParams{
		Scheduler: jobScheduler,
		Timeout:   config.SchedulerStopTimeout,
		Logger:    slog.Default(),
	})

	slog.Info("scheduler started")

	dependencies, err := s.BuildRouterDependencies(buildRouterDependenciesParams{
		Database:       db,
		EventPublisher: broker.Publisher,
		JWTSecret:      cfg.JWTSecret,
		AppEnv:         cfg.AppEnv,
		FileStorage:    fileStorage.FileStorage,
		ImageProcessor: imageProcessor.ImageProcessor,
	})
	if err != nil {
		return err
	}

	listener, err := s.ListenTCP("tcp", cfg.ListenAddress())
	if err != nil {
		return err
	}

	srv := &http.Server{
		Handler:           routes.BuildRouter(dependencies),
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		ReadTimeout:       config.ReadTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
	}

	signalCtx, stopSignalWatch := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignalWatch()

	fmt.Println("\n\x1b[34m [ Server listening ] \x1b[0m")
	fmt.Println("")
	slog.Info("server listening", "address", listener.Addr().String())

	err = httpserver.ServeUntilShutdown(signalCtx, httpserver.ServeUntilShutdownParams{
		Server:          srv,
		Listener:        listener,
		ShutdownTimeout: config.ShutdownTimeout,
		StopSignalWatch: stopSignalWatch,
	})
	if err != nil {
		return err
	}

	fmt.Println("\n\x1b[34m [ Server shut down gracefully ] \x1b[0m")
	fmt.Println("")
	slog.Info("server shut down gracefully")

	return nil
}
