package gin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
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
	CustomTraceHeader  string   // Custom header name for trace ID
	SkipPaths          []string // Paths to skip logging (supports wildcards)
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

// WithSkipPaths configures paths to skip (supports exact match and wildcards)
func WithSkipPaths(paths ...string) Option {
	return func(o *ObservabilityOptions) {
		o.SkipPaths = append(o.SkipPaths, paths...)
	}
}

// WithSkipMetrics is a convenience function to skip common metrics endpoints
func WithSkipMetrics() Option {
	return func(o *ObservabilityOptions) {
		commonPaths := []string{
			"/health",
			"/healthz",
			"/health/*",
			"/ready",
			"/readiness",
			"/live",
			"/liveness",
			"/metrics",
			"/prometheus",
			"/status",
			"/ping",
			"/version",
			"/info",
			"/favicon.ico",
			"/robots.txt",
		}
		o.SkipPaths = append(o.SkipPaths, commonPaths...)
	}
}

// Observability middleware with configurable trace injection
func Observability(opts ...Option) gin.HandlerFunc {
	// Default configuration
	config := &ObservabilityOptions{
		TraceInjectionMode: InjectTraceIDOnly,
		SkipPaths:          []string{"/metrics"}, // Default skip /metrics
	}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	return func(c *gin.Context) {
		start := time.Now()
		ctx := c.Request.Context()

		// Check if this request should be skipped
		shouldSkip := shouldSkipPath(c.Request.URL.Path, c.Request.Method, config.SkipPaths)
		if shouldSkip {
			c.Next()
			return
		}

		// Extract trace information early
		spanCtx := trace.SpanContextFromContext(ctx)

		// Inject trace headers based on configuration (unless skipping tracing)
		injectTraceHeaders(c, spanCtx, config)

		var requestBody string
		var responseBuffer bytes.Buffer

		// Only capture body if we're going to log and debug is enabled
		isDebugLevel := isDebugEnabled()

		if isDebugLevel && c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			requestBody = string(bodyBytes)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		if isDebugLevel {
			writer := &bodyCaptureWriter{ResponseWriter: c.Writer, body: &responseBuffer}
			c.Writer = writer
		}

		c.Next()

		duration := time.Since(start).Seconds()

		// Build structured log
		httpData := map[string]any{
			"request": map[string]any{
				"method": c.Request.Method,
				"path":   c.Request.URL.Path,
				"id":     spanCtx.TraceID().String(),
			},
			"response": map[string]any{
				"status_code": c.Writer.Status(),
			},
		}

		if isDebugLevel {
			httpData["request"].(map[string]any)["body"] = map[string]any{
				"content": requestBody,
				"bytes":   len(requestBody),
			}

			httpData["response"].(map[string]any)["body"] = map[string]any{
				"content": responseBuffer.String(),
				"bytes":   responseBuffer.Len(),
			}
		}

		logLevel := slog.LevelInfo
		if isDebugLevel {
			logLevel = slog.LevelDebug
		}

		slog.Log(ctx, logLevel, "HTTP request completed",
			"event", map[string]any{"duration": duration},
			"source", map[string]any{"ip": c.ClientIP()},
			"http", httpData,
			"user_agent", map[string]any{"original": c.Request.UserAgent()},
		)
	}
}

// shouldSkipPath checks if a path should be skipped based on configuration
func shouldSkipPath(path, method string, skipPaths []string) bool {
	for _, skipPath := range skipPaths {
		if matchPath(path, method, skipPath) {
			return true
		}
	}
	return false
}

// matchPath matches a request path against a skip pattern
func matchPath(requestPath, method, pattern string) bool {
	// Handle method-specific patterns like "GET /metrics"
	if strings.Contains(pattern, " ") {
		parts := strings.SplitN(pattern, " ", 2)
		if len(parts) == 2 {
			patternMethod := strings.ToUpper(strings.TrimSpace(parts[0]))
			patternPath := strings.TrimSpace(parts[1])

			if patternMethod != strings.ToUpper(method) {
				return false
			}
			return matchPathPattern(requestPath, patternPath)
		}
	}

	// Handle path-only patterns
	return matchPathPattern(requestPath, pattern)
}

// matchPathPattern matches a path against a pattern (supports wildcards)
func matchPathPattern(path, pattern string) bool {
	// Exact match
	if path == pattern {
		return true
	}

	// Wildcard support
	if strings.Contains(pattern, "*") {
		return matchWildcard(path, pattern)
	}

	// Prefix match (if pattern ends with /)
	if strings.HasSuffix(pattern, "/") {
		return strings.HasPrefix(path, pattern)
	}

	return false
}

// matchWildcard performs simple wildcard matching
func matchWildcard(text, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// Simple prefix/suffix wildcard matching
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		substr := pattern[1 : len(pattern)-1]
		return strings.Contains(text, substr)
	}

	if strings.HasPrefix(pattern, "*") {
		suffix := pattern[1:]
		return strings.HasSuffix(text, suffix)
	}

	if strings.HasSuffix(pattern, "*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(text, prefix)
	}

	return text == pattern
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

// ObservabilityWithW3CTraceContext creates middleware with W3C trace context
func ObservabilityWithW3CTraceContext() gin.HandlerFunc {
	return Observability(WithTraceInjection(InjectW3CTraceContext))
}

// ObservabilityWithTraceID creates middleware with simple trace ID
func ObservabilityWithTraceID() gin.HandlerFunc {
	return Observability(WithTraceInjection(InjectTraceIDOnly))
}

// ObservabilityWithCustomHeader creates middleware with custom header
func ObservabilityWithCustomHeader(headerName string) gin.HandlerFunc {
	return Observability(
		WithTraceInjection(InjectTraceIDOnly),
		WithCustomTraceHeader(headerName),
	)
}

// ObservabilitySkipMetrics creates middleware that skips common metrics endpoints
func ObservabilitySkipMetrics() gin.HandlerFunc {
	return Observability(WithSkipMetrics())
}

// ObservabilityWithSkipPaths creates middleware with custom skip paths
func ObservabilityWithSkipPaths(paths ...string) gin.HandlerFunc {
	return Observability(WithSkipPaths(paths...))
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
