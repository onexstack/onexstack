package core

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// recordSpanError 是 span 错误记录的核心实现。
// 它统一处理 nil 守卫、错误记录、状态设置与可选的结构化日志。
// message 为空时，回退使用 err.Error()。
func recordSpanError(ctx context.Context, span trace.Span, err error, message string, attrs []attribute.KeyValue, withLog bool) {
	if err == nil {
		return
	}
	if message == "" {
		message = err.Error()
	}

	// 记录错误到 span（带/不带属性）
	if len(attrs) > 0 {
		span.RecordError(err, trace.WithAttributes(attrs...))
	} else {
		span.RecordError(err)
	}

	// 设置 span 状态
	span.SetStatus(codes.Error, message)

	// 可选：记录结构化日志
	if withLog {
		logAttrs := make([]any, 0, len(attrs)*2+2)
		logAttrs = append(logAttrs, "error", err.Error())
		for _, attr := range attrs {
			logAttrs = append(logAttrs, string(attr.Key), attr.Value.AsInterface())
		}
		slog.ErrorContext(ctx, message, logAttrs...)
	}
}

// RecordSpanError 处理 span 错误并添加自定义属性。
func RecordSpanError(ctx context.Context, span trace.Span, err error, attrs ...attribute.KeyValue) {
	recordSpanError(ctx, span, err, "", attrs, false)
}

// RecordSpanErrorWithLog 处理 span 错误、添加自定义属性并记录结构化日志。
func RecordSpanErrorWithLog(ctx context.Context, span trace.Span, err error, message string, attrs ...attribute.KeyValue) {
	recordSpanError(ctx, span, err, message, attrs, true)
}

// LogSpanError 简化版本 - 只需要提供错误，自动使用错误信息作为消息。
func LogSpanError(ctx context.Context, span trace.Span, err error, attrs ...attribute.KeyValue) {
	recordSpanError(ctx, span, err, "", attrs, true)
}

// RecordSpanErrorf 格式化消息版本。
func RecordSpanErrorf(ctx context.Context, span trace.Span, err error, format string, args ...interface{}) {
	recordSpanError(ctx, span, err, fmt.Sprintf(format, args...), nil, true)
}

// RecordSpanErrorfWithAttrs 格式化消息 + 自定义属性版本。
func RecordSpanErrorfWithAttrs(ctx context.Context, span trace.Span, err error, format string, args []interface{}, attrs ...attribute.KeyValue) {
	recordSpanError(ctx, span, err, fmt.Sprintf(format, args...), attrs, true)
}
