package core

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// RecordSpanError 处理span错误并添加自定义属性
func RecordSpanError(ctx context.Context, span trace.Span, err error, attrs ...attribute.KeyValue) {
	// 记录错误到span（带属性）
	if len(attrs) > 0 {
		span.RecordError(err, trace.WithAttributes(attrs...))
	} else {
		span.RecordError(err)
	}

	// 设置span状态
	span.SetStatus(codes.Error, err.Error())
}

// RecordSpanErrorWithLog 处理span错误、添加自定义属性并记录结构化日志
func RecordSpanErrorWithLog(ctx context.Context, span trace.Span, err error, message string, attrs ...attribute.KeyValue) {
	// 1. 记录错误到span（带属性）
	if len(attrs) > 0 {
		span.RecordError(err, trace.WithAttributes(attrs...))
	} else {
		span.RecordError(err)
	}

	// 2. 设置span状态
	span.SetStatus(codes.Error, message)

	// 3. 构建日志属性
	logAttrs := []any{"error", err.Error()}
	for _, attr := range attrs {
		logAttrs = append(logAttrs, string(attr.Key), attr.Value.AsInterface())
	}

	// 4. 记录结构化日志
	slog.ErrorContext(ctx, message, logAttrs...)
}

// LogSpanError 简化版本 - 只需要提供错误，自动使用错误信息作为消息
func LogSpanError(ctx context.Context, span trace.Span, err error, attrs ...attribute.KeyValue) {
	RecordSpanErrorWithLog(ctx, span, err, err.Error(), attrs...)
}

// RecordSpanErrorf 格式化消息版本
func RecordSpanErrorf(ctx context.Context, span trace.Span, err error, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)

	// 记录错误到span
	span.RecordError(err)
	span.SetStatus(codes.Error, message)

	// 记录日志
	slog.ErrorContext(ctx, message, "error", err.Error())
}

// RecordSpanErrorfWithAttrs 格式化消息 + 自定义属性版本
func RecordSpanErrorfWithAttrs(ctx context.Context, span trace.Span, err error, format string, args []interface{}, attrs ...attribute.KeyValue) {
	message := fmt.Sprintf(format, args...)

	// 记录错误到span（带属性）
	if len(attrs) > 0 {
		span.RecordError(err, trace.WithAttributes(attrs...))
	} else {
		span.RecordError(err)
	}
	span.SetStatus(codes.Error, message)

	// 构建日志属性
	logAttrs := []any{"error", err.Error()}
	for _, attr := range attrs {
		logAttrs = append(logAttrs, string(attr.Key), attr.Value.AsInterface())
	}

	// 记录结构化日志
	slog.ErrorContext(ctx, message, logAttrs...)
}
