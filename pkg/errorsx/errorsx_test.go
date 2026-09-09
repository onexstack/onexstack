package errorsx_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"

	"github.com/onexstack/onexstack/pkg/errorsx"
)

func TestErrorX_NewAndToString(t *testing.T) {
	// 创建一个 ErrorX 错误  
	errx := errorsx.New(500, "InternalError.DBConnection", "Database connection failed: %s", "timeout")

	// 检查字段值  
	assert.Equal(t, 500, errx.Code)
	assert.Equal(t, "InternalError.DBConnection", errx.Reason)
	assert.Equal(t, "Database connection failed: timeout", errx.Message)

	// 检查字符串表示  
	expected := `error: code = 500 reason = InternalError.DBConnection message = Database connection failed: timeout metadata = map[]`
	assert.Equal(t, expected, errx.Error())
}

func TestErrorX_WithMessage(t *testing.T) {
	// 创建一个基础错误
	errx := errorsx.New(400, "BadRequest.InvalidInput", "Invalid input for field %s", "username")

	// 更新错误的消息（返回副本，不修改原实例）
	updated := errx.WithMessage("New error message: %s", "retry failed")

	// 验证变更发生在副本上
	assert.Equal(t, "New error message: retry failed", updated.Message)
	assert.Equal(t, 400, updated.Code)                         // Code 不变
	assert.Equal(t, "BadRequest.InvalidInput", updated.Reason) // Reason 不变

	// 原实例保持不变（copy-on-write）
	assert.Equal(t, "Invalid input for field username", errx.Message)
}

func TestErrorX_WithMetadata(t *testing.T) {
	// 创建基础错误
	errx := errorsx.New(400, "BadRequest.InvalidInput", "Invalid input")

	// 添加元数据（返回副本）
	updated := errx.WithMetadata(map[string]string{
		"field": "username",
		"type":  "empty",
	})

	// 验证元数据
	assert.Equal(t, "username", updated.Metadata["field"])
	assert.Equal(t, "empty", updated.Metadata["type"])

	// 动态添加更多元数据（返回副本）
	withKV := updated.KV("user_id", "12345", "trace_id", "xyz-789")
	assert.Equal(t, "12345", withKV.Metadata["user_id"])
	assert.Equal(t, "xyz-789", withKV.Metadata["trace_id"])

	// 原实例未被污染
	assert.Nil(t, errx.Metadata)
}

func TestErrorX_PredefinedNotMutated(t *testing.T) {
	// 全局单例的原始状态
	originalCode := errorsx.ErrBind.Code
	originalMessage := errorsx.ErrBind.Message

	// 调用 mutator 不应改写单例
	_ = errorsx.ErrBind.WithMessage("custom message")
	_ = errorsx.ErrBind.KV("k", "v")

	assert.Equal(t, originalCode, errorsx.ErrBind.Code)
	assert.Equal(t, originalMessage, errorsx.ErrBind.Message)
	assert.Nil(t, errorsx.ErrBind.Metadata)
}

func TestErrorX_WrapAndUnwrap(t *testing.T) {
	cause := errors.New("connection refused")
	errx := errorsx.New(500, "InternalError.DB", "DB failed").Wrap(cause)

	// Unwrap 返回 cause
	assert.Equal(t, cause, errx.Unwrap())

	// errors.Is 沿 cause 链查找
	assert.True(t, errors.Is(errx, cause))

	// errors.Unwrap 也能取到 cause
	assert.Equal(t, cause, errors.Unwrap(errx))

	// 未 Wrap 的错误 Unwrap 返回 nil
	plain := errorsx.New(500, "InternalError.DB", "DB failed")
	assert.Nil(t, plain.Unwrap())
}

func TestErrorX_Is(t *testing.T) {
	// 定义两个预定义错误
	err1 := errorsx.New(404, "NotFound.User", "User not found")
	err2 := errorsx.New(404, "NotFound.User", "Another message")
	err3 := errorsx.New(403, "Forbidden", "Access denied")

	// 验证两个错误均被认为是同一种类型的错误（Code 和 Reason 相等）
	assert.True(t, err1.Is(err2))  // Message 不影响匹配
	assert.False(t, err1.Is(err3)) // Reason 不同
}

func TestErrorX_FromError_WithPlainError(t *testing.T) {
	// 创建一个普通的 Go 错误
	plainErr := errors.New("Something went wrong")

	// 转换为 ErrorX
	errx := errorsx.FromError(plainErr)

	// 检查转换后的 ErrorX
	assert.Equal(t, errorsx.UnknownCode, errx.Code)       // 默认 500
	assert.Equal(t, errorsx.UnknownReason, errx.Reason)   // 默认 ""
	assert.Equal(t, "Something went wrong", errx.Message) // 转换时保留原始错误消息
}

func TestErrorX_FromError_WithGRPCError(t *testing.T) {
	// 创建一个 gRPC 错误
	grpcErr := status.New(3, "Invalid argument").Err() // gRPC INVALID_ARGUMENT = 3

	// 转换为 ErrorX  
	errx := errorsx.FromError(grpcErr)

	// 检查转换后的 ErrorX  
	assert.Equal(t, 400, errx.Code) // httpstatus.FromGRPCCode(3) 对应 HTTP 400  
	assert.Equal(t, "Invalid argument", errx.Message)

	// 没有附加的元数据  
	assert.Nil(t, errx.Metadata)
}

func TestErrorX_FromError_WithGRPCErrorDetails(t *testing.T) {
	// 创建带有详细信息的 gRPC 错误  
	st := status.New(3, "Invalid argument")
	grpcErr, err := st.WithDetails(&errdetails.ErrorInfo{
		Reason:   "InvalidInput",
		Metadata: map[string]string{"field": "name", "type": "required"},
	})
	assert.NoError(t, err) // 确保 gRPC 错误创建成功  

	// 转换为 ErrorX  
	errx := errorsx.FromError(grpcErr.Err())

	// 检查转换后的 ErrorX  
	assert.Equal(t, 400, errx.Code) // gRPC INVALID_ARGUMENT = HTTP 400
	assert.Equal(t, "Invalid argument", errx.Message)
	assert.Equal(t, "InvalidInput", errx.Reason) // 从 gRPC ErrorInfo 中提取  

	// 检查元数据
	assert.Equal(t, "name", errx.Metadata["field"])
	assert.Equal(t, "required", errx.Metadata["type"])
}

func TestCodeAndReasonNil(t *testing.T) {
	// nil 错误的语义：Code 返回 200（无错误），Reason 返回空串（UnknownReason）。
	// 二者口径应一致——nil 不代表"内部错误".
	assert.Equal(t, 200, errorsx.Code(nil))
	assert.Equal(t, "", errorsx.Reason(nil))
	assert.Equal(t, errorsx.UnknownReason, errorsx.Reason(nil))
}

func TestErrorX_WithMetadataDefensiveCopy(t *testing.T) {
	errx := errorsx.New(400, "BadRequest", "bad request")

	md := map[string]string{"field": "username"}
	updated := errx.WithMetadata(md)

	// 修改调用方传入的 map 不应污染错误实例（防御性拷贝）.
	md["field"] = "email"
	md["extra"] = "injected"

	assert.Equal(t, "username", updated.Metadata["field"])
	assert.NotContains(t, updated.Metadata, "extra")
	assert.Len(t, updated.Metadata, 1)
}
