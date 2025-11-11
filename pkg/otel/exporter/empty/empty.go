package tracing

import (
	"context"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Exporter implements sdktrace.SpanExporter.
type Exporter struct{}

// Ensure Exporter implements sdktrace.SpanExporter.
var _ sdktrace.SpanExporter = (*Exporter)(nil)

// ExportSpans logs completed spans using slog.
func (e *Exporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	return nil
}

// Shutdown closes the logger if needed (noop here).
func (e *Exporter) Shutdown(ctx context.Context) error {
	return nil
}

// NewEmptyExporter creates and returns a new instance of Exporter that satisfies
// the OpenTelemetry sdktrace.SpanExporter interface but does not output, store,
// or forward any span data.
func NewEmptyExporter() *Exporter {
	return &Exporter{}
}
