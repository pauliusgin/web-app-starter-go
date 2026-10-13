package main

import (
	"log/slog"

	rbmq "github.com/wagslane/go-rabbitmq"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/brokers/rabbitmq"
)

type openEventBrokerParams struct {
	AmqpURL string
	Logger  *slog.Logger
}

type eventBroker struct {
	Connection *rbmq.Conn
	Publisher  rabbitmq.EventPublisher
}

// Close releases the publisher before the connection.
func (b eventBroker) Close() {
	b.Publisher.Close()
	b.Connection.Close()
}

func openEventBroker(params openEventBrokerParams) (eventBroker, error) {
	logger := params.Logger
	if logger == nil {
		logger = slog.Default()
	}

	connection, err := rabbitmq.NewBrokerConnection(params.AmqpURL, logger)
	if err != nil {
		return eventBroker{}, err
	}

	eventPublisher, err := rabbitmq.NewEventPublisher(rabbitmq.NewEventPublisherParams{
		Connection: connection,
		Logger:     logger,
	})
	if err != nil {
		connection.Close()
		return eventBroker{}, err
	}

	return eventBroker{Connection: connection, Publisher: eventPublisher}, nil
}
