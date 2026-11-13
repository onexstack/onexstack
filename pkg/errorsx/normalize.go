package errorsx

import (
	"context"
	"errors"
)

// Normalize 把任意错误归一化为可以安全写入 HTTP 响应的错误.
//
// HTTP 响应是错误信息的最终出口：把底层错误（数据库、HTTP 客户端、文件系统等）
// 原样序列化给调用方，既会泄露内部路径与连接串，也会让响应丢失 Reason 语义码。
// Normalize 保证非 nil 的返回值一定是 *ErrorX：
//
//   - 已经是 *ErrorX 的错误原样返回，保留调用方设定的状态码与 Reason；
//   - context.DeadlineExceeded 映射为 ErrTimeout；
//   - 其余错误映射为 ErrInternal，原始错误通过 Wrap 保留在错误链上，
//     供日志与 errors.Is/As 使用，但不会出现在响应消息里。
//
// 注意：本函数不感知 gorm、Kubernetes 等具体存储的 sentinel 错误（库不引入这些
// 依赖）。"记录不存在"这类语义必须在错误源头（biz/store 边界）映射成
// ErrNotFound，而不是指望这里猜测。
func Normalize(err error) error {
	if err == nil {
		return nil
	}

	var errx *ErrorX
	if errors.As(err, &errx) {
		return err
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return ErrTimeout.Wrap(err)
	}

	return ErrInternal.Wrap(err)
}
