package onex

import (
	"context"

	"github.com/onexstack/onexstack/pkg/log"
	"github.com/onexstack/onexstack/pkg/logger"
)

// onexLogger provides an implementation of the logger.Logger interface.
type onexLogger struct {
	// preAssignedKVs stores key-value pairs that will be added to every log entry.
	preAssignedKVs []any
}

// Ensure that onexLogger implements the logger.Logger interface.
var _ logger.Logger = (*onexLogger)(nil)

// NewLogger creates a new instance of onexLogger.
func NewLogger() *onexLogger {
	return &onexLogger{}
}

// Debug logs a debug message with any additional key-value pairs.
func (l *onexLogger) Debug(msg string, kvs ...any) {
	log.Debugw(msg, l.mergeKVs(kvs)...)
}

// Warn logs a warning message with any additional key-value pairs.
func (l *onexLogger) Warn(msg string, kvs ...any) {
	log.Warnw(msg, l.mergeKVs(kvs)...)
}

// Info logs an informational message with any additional key-value pairs.
func (l *onexLogger) Info(msg string, kvs ...any) {
	log.Infow(msg, l.mergeKVs(kvs)...)
}

// Error logs an error message with any additional key-value pairs.
func (l *onexLogger) Error(msg string, kvs ...any) {
	log.Errorw(nil, msg, l.mergeKVs(kvs)...)
}

// DebugContext logs a debug message with context and optional key-value pairs.
func (l *onexLogger) DebugContext(ctx context.Context, msg string, kvs ...any) {
	log.W(ctx).Debugw(msg, l.mergeKVs(kvs)...)
}

// WarnContext logs a warning message with context and optional key-value pairs.
func (l *onexLogger) WarnContext(ctx context.Context, msg string, kvs ...any) {
	log.W(ctx).Warnw(msg, l.mergeKVs(kvs)...)
}

// InfoContext logs an informational message with context and optional key-value pairs.
func (l *onexLogger) InfoContext(ctx context.Context, msg string, kvs ...any) {
	log.W(ctx).Infow(msg, l.mergeKVs(kvs)...)
}

// ErrorContext logs an error message with context and optional key-value pairs.
func (l *onexLogger) ErrorContext(ctx context.Context, msg string, kvs ...any) {
	log.W(ctx).Errorw(nil, msg, l.mergeKVs(kvs)...)
}

// With returns a NEW Logger instance with the provided key-value pairs pre-assigned.
// This allows creating child loggers with common fields (e.g., request_id, module_name).
func (l *onexLogger) With(args ...any) logger.Logger {
	return &onexLogger{
		preAssignedKVs: append(l.preAssignedKVs, args...),
	}
}

// mergeKVs merges pre-assigned key-value pairs with the provided ones.
// Pre-assigned kvs come first, so they can be overridden by later kvs if keys conflict.
func (l *onexLogger) mergeKVs(kvs []any) []any {
	if len(l.preAssignedKVs) == 0 {
		return kvs
	}

	merged := make([]any, 0, len(l.preAssignedKVs)+len(kvs))
	merged = append(merged, l.preAssignedKVs...)
	merged = append(merged, kvs...)
	return merged
}
