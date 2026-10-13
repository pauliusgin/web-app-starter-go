package rabbitmq

import (
	"fmt"
	"log/slog"

	rbmq "github.com/wagslane/go-rabbitmq"
)

type brokerLogger struct {
	logger *slog.Logger
}

func newBrokerLogger(logger *slog.Logger) brokerLogger {
	if logger == nil {
		return brokerLogger{logger: slog.Default()}
	}

	return brokerLogger{logger: logger}
}

func (l brokerLogger) Fatalf(format string, v ...any) {
	l.logger.Error(fmt.Sprintf(format, v...))
}

func (l brokerLogger) Errorf(format string, v ...any) {
	l.logger.Error(fmt.Sprintf(format, v...))
}

func (l brokerLogger) Warnf(format string, v ...any) {
	l.logger.Warn(fmt.Sprintf(format, v...))
}

func (l brokerLogger) Infof(format string, v ...any) {
	l.logger.Debug(fmt.Sprintf(format, v...))
}

func (l brokerLogger) Debugf(format string, v ...any) {
	l.logger.Debug(fmt.Sprintf(format, v...))
}

var _ rbmq.Logger = brokerLogger{}
