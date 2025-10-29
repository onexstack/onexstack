package gin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// Standard trace header keys
const (
	// W3C Trace Context standard (most recommended)
	TraceParentHeaderKey = "traceparent"

	// Simple trace ID (most widely used)
	TraceIDHeaderKey = "X-Trace-Id"

	// Generic request ID (universal compatibility)
	RequestIDHeaderKey = "X-Request-Id"

	// Tracestate for additional context
	TraceStateHeaderKey = "tracestate"
)

// TraceInjectionMode defines how trace information is injected
type TraceInjectionMode int

const (
	// InjectW3CTraceContext injects full W3C trace context (recommended)
	InjectW3CTraceContext TraceInjectionMode = iota
	// InjectTraceIDOnly injects only trace ID
	InjectTraceIDOnly
	// InjectBoth injects both W3C format and simple trace ID
	InjectBoth
	// InjectNone disables trace injection
	InjectNone
)

// ObservabilityOptions holds configuration for trace injection
type ObservabilityOptions struct {
	TraceInjectionMode TraceInjectionMode
	CustomTraceHeader  string // Custom header name for trace ID
}

// Option is a functional option for configuring the middleware
type Option func(*ObservabilityOptions)

// WithTraceInjection configures trace injection mode
func WithTraceInjection(mode TraceInjectionMode) Option {
	return func(o *ObservabilityOptions) {
		o.TraceInjectionMode = mode
	}
}

// WithCustomTraceHeader sets a custom header name for trace ID
func WithCustomTraceHeader(headerName string) Option {
	return func(o *ObservabilityOptions) {
		o.CustomTraceHeader = headerName
	}
}

// Observability middleware with configurable trace injection
func Observability(opts ...Option) gin.HandlerFunc {
	// Default configuration
	config := &ObservabilityOptions{
		TraceInjectionMode: InjectTraceIDOnly,
	}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	return func(c *gin.Context) {
		start := time.Now()
		ctx := c.Request.Context()

		// Extract trace information early
		span := trace.SpanFromContext(ctx)
		spanCtx := span.SpanContext()

		// Inject trace headers based on configuration
		injectTraceHeaders(c, spanCtx, config)

		var requestBody string
		if isDebugEnabled() && c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			requestBody = string(bodyBytes)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		var responseBuffer bytes.Buffer
		if isDebugEnabled() {
			writer := &bodyCaptureWriter{ResponseWriter: c.Writer, body: &responseBuffer}
			c.Writer = writer
		}

		c.Next()
		duration := time.Since(start).Seconds()

		// Build structured log
		httpData := map[string]any{
			"request": map[string]any{
				"method": c.Request.Method,
				"route":  c.FullPath(),
			},
			"response": map[string]any{
				"status_code": c.Writer.Status(),
			},
		}

		if isDebugEnabled() {
			httpData["request"].(map[string]any)["body"] = map[string]any{
				"content": requestBody,
				"bytes":   len(requestBody),
			}

			httpData["response"].(map[string]any)["body"] = map[string]any{
				"content": responseBuffer.String(),
				"bytes":   responseBuffer.Len(),
			}
		}

		slog.InfoContext(ctx, "HTTP request completed",
			"duration_sec", duration,
			"source", map[string]any{"ip": c.ClientIP()},
			"http", httpData,
			"user", map[string]any{"agent": c.Request.UserAgent()},
			"trace", map[string]any{"id": spanCtx.TraceID().String()},
			"span", map[string]any{"id": spanCtx.SpanID().String()},
		)
	}
}

// injectTraceHeaders injects trace headers based on configuration
func injectTraceHeaders(c *gin.Context, spanCtx trace.SpanContext, config *ObservabilityOptions) {
	if !spanCtx.IsValid() {
		return
	}

	traceID := spanCtx.TraceID().String()
	spanID := spanCtx.SpanID().String()

	switch config.TraceInjectionMode {
	case InjectW3CTraceContext:
		// W3C Trace Context format: version-trace_id-parent_id-trace_flags
		traceFlags := "01" // sampled
		if !spanCtx.IsSampled() {
			traceFlags = "00" // not sampled
		}
		traceparent := fmt.Sprintf("00-%s-%s-%s", traceID, spanID, traceFlags)
		c.Header(TraceParentHeaderKey, traceparent)

	case InjectTraceIDOnly:
		headerKey := TraceIDHeaderKey
		if config.CustomTraceHeader != "" {
			headerKey = config.CustomTraceHeader
		}
		c.Header(headerKey, traceID)

	case InjectBoth:
		// W3C format
		traceFlags := "01"
		if !spanCtx.IsSampled() {
			traceFlags = "00"
		}
		traceparent := fmt.Sprintf("00-%s-%s-%s", traceID, spanID, traceFlags)
		c.Header(TraceParentHeaderKey, traceparent)

		// Simple trace ID
		headerKey := TraceIDHeaderKey
		if config.CustomTraceHeader != "" {
			headerKey = config.CustomTraceHeader
		}
		c.Header(headerKey, traceID)

	case InjectNone:
		// Do nothing
	}
}

// Convenience functions for common configurations
func ObservabilityWithW3CTraceContext() gin.HandlerFunc {
	return Observability(WithTraceInjection(InjectW3CTraceContext))
}

func ObservabilityWithTraceID() gin.HandlerFunc {
	return Observability(WithTraceInjection(InjectTraceIDOnly))
}

func ObservabilityWithCustomHeader(headerName string) gin.HandlerFunc {
	return Observability(
		WithTraceInjection(InjectTraceIDOnly),
		WithCustomTraceHeader(headerName),
	)
}

// bodyCaptureWriter captures and duplicates written response body
type bodyCaptureWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyCaptureWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// isDebugEnabled checks if debug logging is enabled for the global logger
func isDebugEnabled() bool {
	return slog.Default().Enabled(context.Background(), slog.LevelDebug)
}
