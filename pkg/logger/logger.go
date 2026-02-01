package logger

import "context"

// Logger is a unified interface for structured logging.
// It supports both context-aware and context-free logging methods,
// as well as attribute accumulation.
type Logger interface {
	// Debug logs a message at the debug level with optional key-value pairs.
	Debug(message string, keysAndValues ...any)

	// Warn logs a message at the warning level with optional key-value pairs.
	Warn(message string, keysAndValues ...any)

	// Info logs a message at the info level with optional key-value pairs.
	Info(message string, keysAndValues ...any)

	// Error logs a message at the error level with optional key-value pairs.
	Error(message string, keysAndValues ...any)

	// Debug logs a message at the debug level with context and optional key-value pairs.
	DebugContext(ctx context.Context, message string, keysAndValues ...any)

	// Warn logs a message at the warning level with context and optional key-value pairs.
	WarnContext(ctx context.Context, message string, keysAndValues ...any)

	// Info logs a message at the info level with context and optional key-value pairs.
	InfoContext(ctx context.Context, message string, keysAndValues ...any)

	// Error logs a message at the error level with context and optional key-value pairs.
	ErrorContext(ctx context.Context, message string, keysAndValues ...any)

	// With returns a NEW Logger instance with the provided key-value pairs pre-assigned.
	// This allows creating child loggers with common fields (e.g., request_id, module_name).
	With(args ...any) Logger
}
