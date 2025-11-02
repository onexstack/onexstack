package breeze

import (
	"context"
	"log/slog"
)

// Logger is a Breeze logger implementation using log/slog.
type Logger struct {
	logger *slog.Logger
}

// NewLogger creates a new logger using the default slog instance.
func NewLogger() *Logger {
	return &Logger{logger: slog.Default()}
}

// Debug outputs debug information.
func (l *Logger) Debug(ctx context.Context, msg string, args ...any) {
	l.logger.DebugContext(ctx, msg, args...)
}

// Info outputs general information.
func (l *Logger) Info(ctx context.Context, msg string, args ...any) {
	l.logger.InfoContext(ctx, msg, args...)
}

// Warn outputs warning information.
func (l *Logger) Warn(ctx context.Context, msg string, args ...any) {
	l.logger.WarnContext(ctx, msg, args...)
}

// Error outputs error information.
func (l *Logger) Error(ctx context.Context, msg string, args ...any) {
	l.logger.ErrorContext(ctx, msg, args...)
}
