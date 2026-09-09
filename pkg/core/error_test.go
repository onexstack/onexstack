package core

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

// newNoopSpan 返回一个不产生任何效果的 span，用于测试.
func newNoopSpan() trace.Span {
	_, span := trace.NewNoopTracerProvider().Tracer("test").Start(context.Background(), "test")
	return span
}

func TestRecordSpanError_NilErrNotPanic(t *testing.T) {
	ctx := context.Background()

	// 这些函数对 nil err 都不应 panic.
	RecordSpanError(ctx, newNoopSpan(), nil)
	RecordSpanErrorWithLog(ctx, newNoopSpan(), nil, "msg")
	LogSpanError(ctx, newNoopSpan(), nil)
	RecordSpanErrorf(ctx, newNoopSpan(), nil, "format %s", "x")
	RecordSpanErrorfWithAttrs(ctx, newNoopSpan(), nil, "format %s", []interface{}{"x"})
}

func TestRecordSpanErrorf_Message(t *testing.T) {
	ctx := context.Background()
	err := errors.New("boom")

	// 不应 panic；消息被正确格式化（无断言，仅验证执行路径不崩溃）。
	RecordSpanErrorf(ctx, newNoopSpan(), err, "failed %s", "db")
	RecordSpanError(ctx, newNoopSpan(), err)
	LogSpanError(ctx, newNoopSpan(), err)
}
