package watch

import (
	"log/slog"
)

// cronLogger implement the cron.Logger interface.
type cronLogger struct{}

// NewLogger returns a cron logger.
func NewLogger() *cronLogger {
	return &cronLogger{}
}

// Debug logs routine messages about cron's operation.
func (l *cronLogger) Debug(msg string, kvs ...any) {
	slog.Debug(msg, kvs...)
}

// Info logs routine messages about cron's operation.
func (l *cronLogger) Info(msg string, kvs ...any) {
	slog.Debug(msg, kvs...)
}

// Error logs an error condition.
func (l *cronLogger) Error(err error, msg string, kvs ...any) {
	// 将error作为第一个键值对参数添加
	args := make([]any, 0, len(kvs)+2)
	args = append(args, "error", err)
	args = append(args, kvs...)
	slog.Error(msg, args...)
}
