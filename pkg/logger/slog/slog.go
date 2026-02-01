package slog

import (
	"context"
	"log/slog"

	"github.com/onexstack/onexstack/pkg/logger"
)

// Ensure SlogAdapter implements the Logger interface at compile time.
var _ logger.Logger = (*SlogAdapter)(nil)

// SlogAdapter is a wrapper around the standard library's *slog.Logger
// that implements the Logger interface.
type SlogAdapter struct {
	inner *slog.Logger
}

// Default returns a new Logger instance backed by the standard library's
// global default logger (slog.Default()).
func Default() logger.Logger {
	return &SlogAdapter{inner: slog.Default()}
}

// NewSlogAdapter creates a new Logger backed by the provided *slog.Logger.
// If l is nil, it falls back to using slog.Default().
func NewSlogAdapter(l *slog.Logger) logger.Logger {
	if l == nil {
		l = slog.Default()
	}
	return &SlogAdapter{inner: l}
}

// Debug logs a message at the Debug level.
func (s *SlogAdapter) Debug(msg string, args ...any) {
	s.inner.Debug(msg, args...)
}

// Info logs a message at the Info level.
func (s *SlogAdapter) Info(msg string, args ...any) {
	s.inner.Info(msg, args...)
}

// Warn logs a message at the Warn level.
func (s *SlogAdapter) Warn(msg string, args ...any) {
	s.inner.Warn(msg, args...)
}

// Error logs a message at the Error level.
func (s *SlogAdapter) Error(msg string, args ...any) {
	s.inner.Error(msg, args...)
}

// DebugContext logs a message at the Debug level with the given context.
func (s *SlogAdapter) DebugContext(ctx context.Context, msg string, args ...any) {
	s.inner.DebugContext(ctx, msg, args...)
}

// InfoContext logs a message at the Info level with the given context.
func (s *SlogAdapter) InfoContext(ctx context.Context, msg string, args ...any) {
	s.inner.InfoContext(ctx, msg, args...)
}

// WarnContext logs a message at the Warn level with the given context.
func (s *SlogAdapter) WarnContext(ctx context.Context, msg string, args ...any) {
	s.inner.WarnContext(ctx, msg, args...)
}

// ErrorContext logs a message at the Error level with the given context.
func (s *SlogAdapter) ErrorContext(ctx context.Context, msg string, args ...any) {
	s.inner.ErrorContext(ctx, msg, args...)
}

// With returns a new Logger instance with the provided attributes pre-assigned.
// The returned Logger is a child of the current Logger and shares its handler.
func (s *SlogAdapter) With(args ...any) logger.Logger {
	return &SlogAdapter{
		inner: s.inner.With(args...),
	}
}
