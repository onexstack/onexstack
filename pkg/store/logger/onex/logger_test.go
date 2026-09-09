package onex

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

// storeLikeError simulates the onexstack/pkg/store Store method that calls
// logger.Error. The //go:noinline directive keeps the frame skip deterministic
// so the test can assert on the source location.
//
//go:noinline
func storeLikeError(l *onexLogger, ctx context.Context) {
	l.Error(ctx, errors.New("boom"), "store op failed", "key", "value")
}

// TestErrorReportsCallerSource verifies that the store logger reports the
// application call site rather than its own source file.
func TestErrorReportsCallerSource(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{AddSource: true})

	old := slog.Default()
	slog.SetDefault(slog.New(h))
	defer slog.SetDefault(old)

	storeLikeError(NewLogger(), context.Background())

	out := buf.String()
	if strings.Contains(out, "logger.go") {
		t.Fatalf("source should not point into the store logger; got:\n%s", out)
	}
	if !strings.Contains(out, "logger_test.go") {
		t.Fatalf("source should point to the test caller; got:\n%s", out)
	}
	if !strings.Contains(out, "store op failed") {
		t.Fatalf("message should be logged; got:\n%s", out)
	}
}
