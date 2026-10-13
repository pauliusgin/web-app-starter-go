package scheduler

import (
	"log/slog"

	"github.com/robfig/cron/v3"
)

type cronLogger struct {
	logger *slog.Logger
}

func newCronLogger(logger *slog.Logger) cronLogger {
	if logger == nil {
		return cronLogger{logger: slog.Default()}
	}

	return cronLogger{logger: logger}
}

func (l cronLogger) Info(msg string, keysAndValues ...any) {
	l.logger.Debug(msg, keysAndValues...)
}

func (l cronLogger) Error(err error, msg string, keysAndValues ...any) {
	l.logger.Error(msg, append([]any{"error", err}, keysAndValues...)...)
}

var _ cron.Logger = cronLogger{}
