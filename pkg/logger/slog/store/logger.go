package store

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

// Logger implements the store.Logger interface using log/slog. It resolves the
// application call site that triggered the store operation and bakes it into the
// record, so the reported file:line and function point at the caller's own code
// rather than this shared library.
type Logger struct{}

// NewLogger creates and returns a new instance of Logger.
func NewLogger() *Logger {
	return &Logger{}
}

// Error logs an error message with the provided context using log/slog.
//
// The source location is resolved three frames above this method — past this
// method and the onexstack/pkg/store Store method that invoked it — so the
// reported file:line and function identify the application's call site.
// The //go:noinline directive below keeps that frame skip deterministic.
//
//go:noinline
func (l *Logger) Error(ctx context.Context, err error, msg string, kvs ...any) {
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
