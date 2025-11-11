package tracing

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// FakeExporter implements sdktrace.SpanExporter but discards or logs spans.
// You can pass a custom handler to inspect spans in tests.
type FakeExporter struct {
	LogSpans bool                                                  // if true, print spans to stdout
	Handle   func(ctx context.Context, span sdktrace.ReadOnlySpan) // optional span handler
}

// Ensure FakeExporter implements sdktrace.SpanExporter.
var _ sdktrace.SpanExporter = (*FakeExporter)(nil)

// ExportSpans is called by the SDK when spans are finished.
// In this fake exporter, we either log them or discard them.
func (f *FakeExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		if f.Handle != nil {
			f.Handle(ctx, span)
		}
		if f.LogSpans {
			fmt.Printf("[FAKE EXPORTER] span finished: name=%s traceID=%s spanID=%s status=%v duration=%v\n",
				span.Name(),
				span.SpanContext().TraceID().String(),
				span.SpanContext().SpanID().String(),
				span.Status(),
				time.Since(span.StartTime()),
			)
		}
	}
	return nil
}

// Shutdown is called when the provider or process exits.
func (f *FakeExporter) Shutdown(ctx context.Context) error {
	if f.LogSpans {
		log.Println("[FAKE EXPORTER] shutting down")
	}
	return nil
}

// InitFakeTracer configures OpenTelemetry to use a fake tracer provider.
// It creates valid TraceIDs/SpanIDs, keeps propagation functional, but doesn't export spans elsewhere.
func InitFakeTracer(serviceName string, logSpans bool) {
	exporter := &FakeExporter{LogSpans: logSpans}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			"fake-resource",
			attribute.String("service.name", serviceName),
			attribute.String("deployment.environment", "development"),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	if logSpans {
		log.Printf("[FAKE EXPORTER] Initialized fake tracer for service=%s", serviceName)
	}
}
