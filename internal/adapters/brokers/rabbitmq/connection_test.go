package rabbitmq

import (
	"errors"
	"io"
	"log/slog"
	"net"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/pauliusgin/web_app_starter-be/internal/config"
)

const localRabbitMQURL = "amqp://user:password@localhost:5672/web_app_starter-dev"

const brokerProbeTimeout = time.Second

func discardingLoggerForTest() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func liveRabbitMQURL() string {
	amqpURL := os.Getenv(config.RabbitMQURLVariable)
	if amqpURL == "" {
		return localRabbitMQURL
	}

	return amqpURL
}

func requireReachableBroker(t *testing.T, amqpURL string) {
	t.Helper()

	parsedURL, err := url.Parse(amqpURL)
	if err != nil {
		t.Fatalf("url.Parse(%q): %v", amqpURL, err)
	}

	probe, err := net.DialTimeout("tcp", parsedURL.Host, brokerProbeTimeout)
	if err != nil {
		t.Skipf("rabbitmq unavailable (run `make mq-start`): %v", err)
	}
	probe.Close()
}

func closedLocalAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	address := listener.Addr().String()
	listener.Close()

	return address
}

func TestNewBrokerConnectionDialsALiveBroker(t *testing.T) {
	amqpURL := liveRabbitMQURL()
	requireReachableBroker(t, amqpURL)

	connection, err := NewBrokerConnection(amqpURL, discardingLoggerForTest())
	if err != nil {
		t.Fatalf("NewBrokerConnection = %v, want a connection", err)
	}
	if connection == nil {
		t.Fatal("NewBrokerConnection returned no connection and no error")
	}

	if err := connection.Close(); err != nil {
		t.Errorf("Close = %v, want nil", err)
	}
}

func TestNewBrokerConnectionReportsAnUnreachableBroker(t *testing.T) {
	unreachableURL := "amqp://user:password@" + closedLocalAddress(t) + "/web_app_starter"

	connection, err := NewBrokerConnection(unreachableURL, discardingLoggerForTest())

	unreachable, ok := errors.AsType[ErrBrokerUnreachable](err)
	if !ok {
		t.Fatalf("NewBrokerConnection = %v, want ErrBrokerUnreachable", err)
	}
	if unreachable.Cause == nil {
		t.Error("ErrBrokerUnreachable.Cause = nil, want the dial error")
	}
	if connection != nil {
		t.Error("a connection was returned alongside the error, want none")
	}
}
