package main

import (
	"context"
	"fmt"
	"log/slog"

	rbmq "github.com/wagslane/go-rabbitmq"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/brokers/rabbitmq"
	"github.com/pauliusgin/web_app_starter-be/internal/api/messaging/consumers"
	"github.com/pauliusgin/web_app_starter-be/internal/api/messaging/handlers/messaging"
	"github.com/pauliusgin/web_app_starter-be/internal/config"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/gateways/email"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/repositories"
	"github.com/pauliusgin/web_app_starter-be/internal/domains/messaging/usecases"
	"github.com/pauliusgin/web_app_starter-be/internal/events"
)

type startEventConsumersParams struct {
	Connection        *rbmq.Conn
	AmqpURL           string
	Logger            *slog.Logger
	FrontendBaseURL   string
	EmailGateway      email.Gateway
	MessageRepository repositories.MessageWriteRepository
	EventDispatcher   events.Dispatcher
}

func startEventConsumers(params startEventConsumersParams) (func(), error) {
	registry := consumers.BuildRegistry(buildRegistryDependencies(params))
	running := rabbitmq.RunningEventConsumers{}

	stopAll := func() {
		rabbitmq.StopAllEventConsumers(running, config.ConsumerStopTimeout)
	}

	err := rabbitmq.DeclareDeadLetterTopology(rabbitmq.DeclareDeadLetterTopologyParams{
		AmqpURL: params.AmqpURL,
		Queues:  registryQueues(registry),
	})
	if err != nil {
		return nil, err
	}

	for _, declared := range registry {
		consumer, err := rabbitmq.NewEventConsumer(rabbitmq.NewEventConsumerParams{
			Connection:     params.Connection,
			Logger:         params.Logger,
			Queue:          declared.Queue,
			RoutingKeys:    []string{string(declared.Label)},
			Concurrency:    declared.Concurrency,
			Prefetch:       declared.Prefetch,
			HandlerTimeout: declared.HandlerTimeout,
			Handle:         rabbitmq.EventDeliveryHandler(declared.Handle),
		})
		if err != nil {
			stopAll()
			return nil, err
		}

		running = append(running, rabbitmq.StartEventConsumer(consumer))
		reportConsumerReady(declared)
	}

	return stopAll, nil
}

func registryQueues(registry consumers.Registry) []string {
	queues := make([]string, 0, len(registry))
	for _, declared := range registry {
		queues = append(queues, declared.Queue)
	}

	return queues
}

func buildRegistryDependencies(params startEventConsumersParams) consumers.RegistryDependencies {
	dependencies := consumers.RegistryDependencies{Logger: params.Logger}

	sendVerificationEmailDependencies := usecases.SendVerificationEmailDependencies{
		Gateway:           params.EmailGateway,
		MessageRepository: params.MessageRepository,
		EventDispatcher:   params.EventDispatcher,
		Logger:            params.Logger,
		FrontendBaseURL:   params.FrontendBaseURL,
	}

	dependencies.VerificationEmail = messaging.VerificationEmailDependencies{
		SendVerificationEmail: func(ctx context.Context, sendParams usecases.SendVerificationEmailParams) error {
			return usecases.SendVerificationEmail(ctx, sendVerificationEmailDependencies, sendParams)
		},
		Logger: params.Logger,
	}

	return dependencies
}

func reportConsumerReady(declared consumers.Consumer) {
	fmt.Printf("\n\x1b[34m [ consumer ready: %s ] \x1b[0m\n\n", declared.Queue)
	slog.Info("event consumer started", "queue", declared.Queue, "event", declared.Label)
}
