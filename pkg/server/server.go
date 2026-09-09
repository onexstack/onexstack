// Copyright 2024 孔令飞 <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/miniblog. The professional
// version of this repository is https://github.com/onexstack/onex.

package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Server 定义所有服务器类型的接口.
type Server interface {
	// Run 运行服务器并阻塞，直到服务器停止（例如被 GracefulStop 关停）或出错.
	// 正常关停时应返回 nil，出错时返回具体错误.
	Run(ctx context.Context) error
	// GracefulStop 优雅关停服务器，需处理 context 的超时时间.
	GracefulStop(ctx context.Context) error
}

// Serve starts the server and blocks until the context is canceled or the
// server exits with an error. When the context is canceled, it gracefully shuts
// the server down within a 10-second timeout budget.
func Serve(ctx context.Context, srv Server) error {
	// 在后台运行 server，捕获其返回的错误。缓冲为 1，避免 goroutine 泄漏.
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Run(ctx)
	}()

	// 阻塞直到 ctx 取消（正常关停）或 server 主动退出（出错或自行停止）.
	select {
	case err := <-errCh:
		// server 在 ctx 取消前就已退出.
		return err
	case <-ctx.Done():
		// 收到关停信号，继续执行优雅关停.
	}

	slog.Info("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Gracefully stop the server.
	if err := srv.GracefulStop(shutdownCtx); err != nil {
		slog.Error("failed to gracefully stop server", "err", err)
	}

	// 等待 Run 返回（通常因 GracefulStop 而返回 nil）.
	if err := <-errCh; err != nil {
		return err
	}

	slog.Info("server exited successfully.")

	return nil
}

// protocolName 从 http.Server 中获取协议名.
func protocolName(server *http.Server) string {
	if server.TLSConfig != nil {
		return "https"
	}
	return "http"
}
