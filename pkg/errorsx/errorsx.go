package errorsx

import (
	"errors"
	"fmt"
	"net/http"

	httpstatus "github.com/go-kratos/kratos/v2/transport/http/status"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"
)

// ErrorX 定义了 OneX 项目体系中使用的错误类型，用于描述错误的详细信息.
type ErrorX struct {
	// Code 表示错误的 HTTP 状态码，用于与客户端进行交互时标识错误的类型.
	Code int `json:"code,omitempty"`

	// Reason 表示错误发生的原因，通常为业务错误码，用于精准定位问题.
	Reason string `json:"reason,omitempty"`

	// Message 表示简短的错误信息，通常可直接暴露给用户查看.
	Message string `json:"message,omitempty"`

	// Metadata 用于存储与该错误相关的额外元信息，可以包含上下文或调试信息.
	Metadata map[string]string `json:"metadata,omitempty"`

	// cause 是底层错误，用于支持 errors.Is/As 沿错误链穿透（不参与 JSON 序列化）.
	cause error
}

// New 创建一个新的错误.
func New(code int, reason string, format string, args ...any) *ErrorX {
	return &ErrorX{
		Code:    code,
		Reason:  reason,
		Message: fmt.Sprintf(format, args...),
	}
}

// Error 实现 error 接口中的 `Error` 方法.
func (err *ErrorX) Error() string {
	return fmt.Sprintf("error: code = %d reason = %s message = %s metadata = %v", err.Code, err.Reason, err.Message, err.Metadata)
}

// clone 返回接收者的浅拷贝（深拷贝 Metadata map），使 mutator 不修改原始实例，
// 从而避免包级单例（如 ErrBind）被并发改写产生数据竞争或错误消息泄漏.
func (err *ErrorX) clone() *ErrorX {
	c := *err
	if err.Metadata != nil {
		c.Metadata = make(map[string]string, len(err.Metadata))
		for k, v := range err.Metadata {
			c.Metadata[k] = v
		}
	}
	return &c
}

// WithMessage 返回一个 Message 字段被更新的副本，不修改原始实例.
func (err *ErrorX) WithMessage(format string, args ...any) *ErrorX {
	c := err.clone()
	c.Message = fmt.Sprintf(format, args...)
	return c
}

// WithMetadata 返回一个 Metadata 字段被设置的副本，不修改原始实例.
// 传入的 map 会被深拷贝，避免调用方后续修改 md 污染本实例（与 copy-on-write 语义一致）.
func (err *ErrorX) WithMetadata(md map[string]string) *ErrorX {
	c := err.clone()
	if md != nil {
		c.Metadata = make(map[string]string, len(md))
		for k, v := range md {
			c.Metadata[k] = v
		}
	}
	return c
}

// KV 返回一个使用 key-value 对设置元数据的副本，不修改原始实例.
func (err *ErrorX) KV(kvs ...string) *ErrorX {
	c := err.clone()
	if c.Metadata == nil {
		c.Metadata = make(map[string]string) // 初始化元数据映射
	}

	for i := 0; i < len(kvs); i += 2 {
		// kvs 必须是成对的
		if i+1 < len(kvs) {
			c.Metadata[kvs[i]] = kvs[i+1]
		}
	}
	return c
}

// GRPCStatus 返回 gRPC 状态表示.
func (err *ErrorX) GRPCStatus() *status.Status {
	details := errdetails.ErrorInfo{Reason: err.Reason, Metadata: err.Metadata}
	s, _ := status.New(httpstatus.ToGRPCCode(err.Code), err.Message).WithDetails(&details)
	return s
}

// WithRequestID 返回一个携带请求 ID 的副本.
func (err *ErrorX) WithRequestID(requestID string) *ErrorX {
	return err.KV("X-Request-ID", requestID) // 设置请求 ID
}

// Wrap 返回一个携带底层 cause 的副本，使本错误可沿错误链穿透.
func (err *ErrorX) Wrap(cause error) *ErrorX {
	c := err.clone()
	c.cause = cause
	return c
}

// Unwrap 返回底层 cause，用于支持 errors.Is/As.
func (err *ErrorX) Unwrap() error {
	return err.cause
}

// Is 判断当前错误是否与目标错误匹配.
// 若目标也是 *ErrorX，则比较 Code 和 Reason 字段（均相等则匹配）；
// 否则沿 cause 链继续查找.
func (err *ErrorX) Is(target error) bool {
	if targetX := new(ErrorX); errors.As(target, &targetX) {
		return targetX.Code == err.Code && targetX.Reason == err.Reason
	}
	return errors.Is(err.cause, target)
}

// Code 返回错误的 HTTP 代码.
func Code(err error) int {
	if err == nil {
		return http.StatusOK //nolint:mnd
	}
	return FromError(err).Code
}

// Reason 返回特定错误的原因.
// 当 err 为 nil 时返回 UnknownReason（空串），与 Code(nil) 返回 200 的语义对齐：
// nil 错误不代表"内部错误"，而是"无错误".
func Reason(err error) string {
	if err == nil {
		return UnknownReason
	}
	return FromError(err).Reason
}

// FromError 尝试将一个通用的 error 转换为自定义的 *ErrorX 类型.
func FromError(err error) *ErrorX {
	// 如果传入的错误是 nil，则直接返回 nil，表示没有错误需要处理.
	if err == nil {
		return nil
	}

	// 检查传入的 error 是否已经是 ErrorX 类型的实例.
	// 如果错误可以通过 errors.As 转换为 *ErrorX 类型，则直接返回该实例.
	if errx := new(ErrorX); errors.As(err, &errx) {
		return errx
	}

	// gRPC 的 status.FromError 方法尝试将 error 转换为 gRPC 错误的 status 对象.
	// 如果 err 不能转换为 gRPC 错误（即不是 gRPC 的 status 错误），
	// 则返回一个带有默认值的 ErrorX，表示是一个未知类型的错误.
	gs, ok := status.FromError(err)
	if !ok {
		return &ErrorX{Code: UnknownCode, Reason: UnknownReason, Message: err.Error()}
	}

	// 如果 err 是 gRPC 的错误类型，会成功返回一个 gRPC status 对象（gs）.
	// 使用 gRPC 状态中的错误代码和消息创建一个 ErrorX.
	ret := &ErrorX{Code: httpstatus.FromGRPCCode(gs.Code()), Reason: UnknownReason, Message: gs.Message()}

	// 遍历 gRPC 错误详情中的所有附加信息（Details）.
	for _, detail := range gs.Details() {
		if typed, ok := detail.(*errdetails.ErrorInfo); ok {
			ret.Reason = typed.Reason
			ret.Metadata = typed.Metadata
			return ret
		}
	}

	return ret
}
