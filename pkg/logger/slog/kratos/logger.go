package store

import (
	"log/slog"
	"os"

	krtlog "github.com/go-kratos/kratos/v2/log"
)

// Logger is a logger that implements the Logger interface.
// It uses the log package to log error messages with additional context.
type Logger struct{}

// NewLogger creates and returns a new instance of Logger.
func NewLogger() *Logger {
	return &Logger{}
}

// Error logs an error message with the provided context using the log package.
// func (l *Logger) Log(ctx context.Context, err error, msg string, kvs ...any) {
func (l *Logger) Log(level krtlog.Level, keyvals ...any) error {
	switch level {
	case krtlog.LevelDebug:
		slog.Debug("", keyvals...)
	case krtlog.LevelInfo:
		slog.Info("", keyvals...)
	case krtlog.LevelWarn:
		slog.Warn("", keyvals...)
	case krtlog.LevelError:
		slog.Error("", keyvals...)
	case krtlog.LevelFatal:
		slog.Error("", keyvals...)
		os.Exit(1)
	}

	return nil
}
