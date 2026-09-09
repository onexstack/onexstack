package onex

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

// onexLogger is a logger that implements the store.Logger interface.
// It uses log/slog to log error messages with additional context.
type onexLogger struct{}

// NewLogger creates and returns a new instance of onexLogger.
func NewLogger() *onexLogger {
	return &onexLogger{}
}

// Error logs an error message with the provided context using log/slog.
//
// The source location is resolved three frames above this method — past this
// method and the onexstack/pkg/store Store method that invoked it — so the
// reported file:line and function identify the application's call site.
// The //go:noinline directive below keeps that frame skip deterministic.
//
//go:noinline
func (l *onexLogger) Error(ctx context.Context, err error, msg string, kvs ...any) {
	kvs = append(kvs, "error", err)

	logger := slog.Default()
	if !logger.Enabled(ctx, slog.LevelError) {
		return
	}

	// skip runtime.Callers, this method, and the Store method that called it.
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:])

	rec := slog.NewRecord(time.Now(), slog.LevelError, msg, pcs[0])
	rec.Add(kvs...)
	_ = logger.Handler().Handle(ctx, rec)
}
