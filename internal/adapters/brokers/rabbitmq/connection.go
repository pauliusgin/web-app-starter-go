package rabbitmq

import (
	"log/slog"

	rbmq "github.com/wagslane/go-rabbitmq"
)

// NewBrokerConnection dials the broker eagerly and returns a connection the caller must close once.
func NewBrokerConnection(amqpURL string, logger *slog.Logger) (*rbmq.Conn, error) {
	connection, err := rbmq.NewConn(
		amqpURL,
		rbmq.WithConnectionOptionsLogger(newBrokerLogger(logger)),
	)
	if err != nil {
		return nil, ErrBrokerUnreachable{Cause: err}
	}

	return connection, nil
}
