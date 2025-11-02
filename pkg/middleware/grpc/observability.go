package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// --- Trace Header Constants ---
const (
	TraceParentHeaderKey = "traceparent"
	TraceIDHeaderKey     = "X-Trace-Id"
	RequestIDHeaderKey   = "X-Request-Id"
	TraceStateHeaderKey  = "tracestate"
)

// --- Trace Injection Mode ---
type TraceInjectionMode int

const (
	InjectW3CTraceContext TraceInjectionMode = iota
	InjectTraceIDOnly
	InjectBoth
	InjectNone
)

// --- Configuration ---
type ObservabilityOptions struct {
	TraceInjectionMode TraceInjectionMode
	CustomTraceHeader  string
}

// --- Option Pattern ---
type Option func(*ObservabilityOptions)

func WithTraceInjection(mode TraceInjectionMode) Option {
	return func(o *ObservabilityOptions) { o.TraceInjectionMode = mode }
}

func WithCustomTraceHeader(header string) Option {
	return func(o *ObservabilityOptions) { o.CustomTraceHeader = header }
}

// --- Main Interceptor ---
func Observability(opts ...Option) grpc.UnaryServerInterceptor {
	cfg := &ObservabilityOptions{TraceInjectionMode: InjectTraceIDOnly}
	for _, opt := range opts {
		opt(cfg)
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()
		spanCtx := trace.SpanFromContext(ctx).SpanContext()

		// Inject trace headers
		md, _ := metadata.FromIncomingContext(ctx)
		md = injectTraceMetadata(ctx, md, spanCtx, cfg)
		ctx = metadata.NewIncomingContext(ctx, md)

		// Optional: capture request payload
		var requestBody string
		if isDebugEnabled() && req != nil {
			data, _ := json.Marshal(req)
			requestBody = string(data)
		}

		// Process RPC
		resp, err := handler(ctx, req)

		// Optional: capture response payload
		var responseBody string
		if isDebugEnabled() && resp != nil {
			data, _ := json.Marshal(resp)
			responseBody = string(data)
		}

		duration := time.Since(start).Seconds()

		// Peer Info
		var clientIP string
		if p, ok := peer.FromContext(ctx); ok {
			clientIP = p.Addr.String()
		}

		status := "OK"
		if err != nil {
			status = "ERROR"
		}

		// Build structured log
		rpcData := map[string]any{
			"request": map[string]any{
				"method": info.FullMethod,
			},
			"response": map[string]any{
				"status": status,
			},
		}

		if isDebugEnabled() {
			rpcData["request"] = map[string]any{
				"body": map[string]any{
					"content": requestBody,
					"bytes":   len(requestBody),
				},
			}
			rpcData["response"] = map[string]any{
				"body": map[string]any{
					"content": responseBody,
					"bytes":   len(responseBody),
				},
			}
		}

		slog.InfoContext(ctx, "gRPC request completed",
			"duration_sec", duration,
			"source", map[string]any{"ip": clientIP},
			"rpc", rpcData,
			"trace", map[string]any{"id": spanCtx.TraceID().String()},
			"span", map[string]any{"id": spanCtx.SpanID().String()},
		)

		return resp, err
	}
}

// --- Inject Trace Metadata ---
func injectTraceMetadata(ctx context.Context, md metadata.MD, spanCtx trace.SpanContext, cfg *ObservabilityOptions) metadata.MD {
	if !spanCtx.IsValid() {
		return md
	}

	traceID := spanCtx.TraceID().String()
	spanID := spanCtx.SpanID().String()

	switch cfg.TraceInjectionMode {
	case InjectW3CTraceContext:
		traceFlags := "01"
		if !spanCtx.IsSampled() {
			traceFlags = "00"
		}
		md.Set(TraceParentHeaderKey, fmt.Sprintf("00-%s-%s-%s", traceID, spanID, traceFlags))

	case InjectTraceIDOnly:
		header := TraceIDHeaderKey
		if cfg.CustomTraceHeader != "" {
			header = cfg.CustomTraceHeader
		}
		md.Set(header, traceID)

	case InjectBoth:
		traceFlags := "01"
		if !spanCtx.IsSampled() {
			traceFlags = "00"
		}
		md.Set(TraceParentHeaderKey, fmt.Sprintf("00-%s-%s-%s", traceID, spanID, traceFlags))
		header := TraceIDHeaderKey
		if cfg.CustomTraceHeader != "" {
			header = cfg.CustomTraceHeader
		}
		md.Set(header, traceID)

	case InjectNone:
		// no-op
	}

	// 将请求 ID 设置到响应的 Header Metadata 中
	// grpc.SetHeader 会在 gRPC 方法响应中添加元数据（Metadata），
	// 此处将包含请求 ID 的 Metadata 设置到 Header 中。
	// 注意：grpc.SetHeader 仅设置数据，它不会立即发送给客户端。
	// Header Metadata 会在 RPC 响应返回时一并发送。
	_ = grpc.SetHeader(ctx, md)

	return md
}

func isDebugEnabled() bool {
	return slog.Default().Enabled(context.Background(), slog.LevelDebug)
}

// --- Convenience Wrappers ---
func ObservabilityWithW3CTraceContext() grpc.UnaryServerInterceptor {
	return Observability(WithTraceInjection(InjectW3CTraceContext))
}

func ObservabilityWithTraceID() grpc.UnaryServerInterceptor {
	return Observability(WithTraceInjection(InjectTraceIDOnly))
}

func ObservabilityWithCustomHeader(headerName string) grpc.UnaryServerInterceptor {
	return Observability(
		WithTraceInjection(InjectTraceIDOnly),
		WithCustomTraceHeader(headerName),
	)
}
