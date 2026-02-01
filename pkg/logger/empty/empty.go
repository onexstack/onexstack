package empty

import (
	"context"

	"github.com/onexstack/onexstack/pkg/logger"
)

// EmptyLogger is a no-op (no-operation) implementation of the Logger interface.
// It discards all log messages and attributes without producing any output.
// This is useful for testing or when logging needs to be disabled.
type EmptyLogger struct{}

// NewLogger returns a new instance of EmptyLogger.
func NewLogger() logger.Logger {
	return &EmptyLogger{}
}

// Ensure EmptyLogger fully implements the Logger interface at compile time.
var _ logger.Logger = (*EmptyLogger)(nil)

// Debug implements the Logger interface.
// It effectively ignores the log message and attributes.
func (l *EmptyLogger) Debug(_ string, _ ...any) {
	// no-op
}

// Warn implements the Logger interface.
// It effectively ignores the log message and attributes.
func (l *EmptyLogger) Warn(_ string, _ ...any) {
	// no-op
}

// Info implements the Logger interface.
// It effectively ignores the log message and attributes.
func (l *EmptyLogger) Info(_ string, _ ...any) {
	// no-op
}

// Error implements the Logger interface.
// It effectively ignores the log message and attributes.
func (l *EmptyLogger) Error(_ string, _ ...any) {
	// no-op
}

// DebugContext implements the Logger interface.
// It ignores the context, message, and attributes.
func (l *EmptyLogger) DebugContext(_ context.Context, _ string, _ ...any) {
	// no-op
}

// WarnContext implements the Logger interface.
// It ignores the context, message, and attributes.
func (l *EmptyLogger) WarnContext(_ context.Context, _ string, _ ...any) {
	// no-op
}

// InfoContext implements the Logger interface.
// It ignores the context, message, and attributes.
func (l *EmptyLogger) InfoContext(_ context.Context, _ string, _ ...any) {
	// no-op
}

// ErrorContext implements the Logger interface.
// It ignores the context, message, and attributes.
func (l *EmptyLogger) ErrorContext(_ context.Context, _ string, _ ...any) {
	// no-op
}

// With implements the Logger interface.
// It returns the current logger instance unchanged, as accumulating attributes
// on a no-op logger has no effect and allocating a new struct is unnecessary.
func (l *EmptyLogger) With(_ ...any) logger.Logger {
	return l
}
