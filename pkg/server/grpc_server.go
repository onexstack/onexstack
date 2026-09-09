// Copyright 2024 孔令飞 <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/miniblog. The professional
// version of this repository is https://github.com/onexstack/onex.

package server

import (
	"context"
	"errors"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	genericoptions "github.com/onexstack/onexstack/pkg/options"
)

// GRPCServer 代表一个 GRPC 服务器.
type GRPCServer struct {
	srv          *grpc.Server
	lis          net.Listener
	healthServer *health.Server
}

// NewGRPCServer 创建一个新的 GRPC 服务器实例.
func NewGRPCServer(
	grpcOptions *genericoptions.GRPCOptions,
	tlsOptions *genericoptions.TLSOptions,
	serverOptions []grpc.ServerOption,
	registerBuilder func() (func(grpc.ServiceRegistrar), string),
) (*GRPCServer, error) {
	lis, err := net.Listen("tcp", grpcOptions.Addr)
	if err != nil {
		slog.Error("failed to listen", "error", err)
		return nil, err
	}

	serverOptions = appendTLSCreds(serverOptions, tlsOptions)

	grpcsrv := grpc.NewServer(serverOptions...)

	registerFn, serverName := registerBuilder()
	registerFn(grpcsrv)
	healthServer := registerHealthServer(serverName, grpcsrv)
	reflection.Register(grpcsrv)

	return &GRPCServer{
		srv:          grpcsrv,
		lis:          lis,
		healthServer: healthServer,
	}, nil
}

// Run 启动 GRPC 服务器并阻塞直到服务器停止或出错。正常关闭（GracefulStop）会
// 使 Serve 返回 grpc.ErrServerStopped，此时 Run 返回 nil，便于调用方处理优雅关停。
func (s *GRPCServer) Run(ctx context.Context) error {
	slog.Info("start to listening the incoming requests", "protocol", "grpc", "addr", s.lis.Addr().String())
	if err := s.srv.Serve(s.lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		return err
	}
	return nil
}

// GracefulStop 优雅地关闭 GRPC 服务器.
// 它尊重 ctx 的超时：若在 ctx 到期前服务器未能自然完成优雅关闭（例如存在
// 长连接或流式 RPC），则强制调用 Stop 立即终止，避免永久阻塞.
func (s *GRPCServer) GracefulStop(ctx context.Context) error {
	slog.Info("gracefully stop grpc server")

	// 先将服务置为 NOT_SERVING，使负载均衡/探活停止转发新流量.
	if s.healthServer != nil {
		s.healthServer.Shutdown()
	}

	done := make(chan struct{})
	go func() {
		s.srv.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("grpc server gracefully stopped")
		return nil
	case <-ctx.Done():
		slog.Warn("grpc server graceful stop timed out, forcing stop", "err", ctx.Err())
		s.srv.Stop()
		<-done
		return ctx.Err()
	}
}

// appendTLSCreds 在启用 TLS 时为 gRPC server options 追加 TLS 凭据。多个 gRPC
// server 实现（如 GRPCServer、PolarisServer）复用此逻辑，避免重复.
func appendTLSCreds(serverOptions []grpc.ServerOption, tlsOptions *genericoptions.TLSOptions) []grpc.ServerOption {
	if tlsOptions != nil && tlsOptions.Enabled {
		tlsConfig := tlsOptions.MustTLSConfig()
		serverOptions = append(serverOptions, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}
	return serverOptions
}

// registerHealthServer 注册健康检查服务并返回 healthServer 实例，供关停时更新状态.
func registerHealthServer(serverName string, grpcsrv *grpc.Server) *health.Server {
	// 创建健康检查服务实例
	healthServer := health.NewServer()

	// 设定服务的健康状态
	healthServer.SetServingStatus(serverName, grpc_health_v1.HealthCheckResponse_SERVING)

	// 注册健康检查服务
	grpc_health_v1.RegisterHealthServer(grpcsrv, healthServer)

	return healthServer
}
