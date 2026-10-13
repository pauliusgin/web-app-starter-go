package main

import (
	"errors"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/brokers/rabbitmq"
)

const invalidAmqpURL = "amqp://guest:guest@[100::1]:5672/"

func TestOpenEventBrokerReportsAnUnreachableBroker(t *testing.T) {
	silenceDefaultLogger(t)

	broker, err := openEventBroker(openEventBrokerParams{AmqpURL: invalidAmqpURL})

	if _, ok := errors.AsType[rabbitmq.ErrBrokerUnreachable](err); !ok {
		t.Fatalf("openEventBroker = %v, want ErrBrokerUnreachable", err)
	}

	if broker.Connection != nil {
		t.Error("a connection was returned alongside the error, want none")
	}

	broker.Publisher.Close()
}
