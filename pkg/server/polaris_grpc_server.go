// Copyright 2024 孔令飞 <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/miniblog. The professional
// version of this repository is https://github.com/onexstack/onex.

package server

import (
	"context"
	"log/slog"
	"net"
	"time"

	polarisgrpc "github.com/polarismesh/grpc-go-polaris"
	"github.com/polarismesh/polaris-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/reflection"

	genericoptions "github.com/onexstack/onexstack/pkg/options"
)

// PolarisServer 代表一个 GRPC 服务器.
type PolarisServer struct {
	srv          *polarisgrpc.Server
	lis          net.Listener
	healthServer *health.Server
}

// NewPolarisServer 创建一个新的 GRPC 服务器实例.
func NewPolarisServer(
	polarisOptions *genericoptions.PolarisOptions,
	grpcOptions *genericoptions.GRPCOptions,
	tlsOptions *genericoptions.TLSOptions,
	serverOptions []grpc.ServerOption,
	registerBuilder func() (func(grpc.ServiceRegistrar), string),
) (*PolarisServer, error) {
	lis, err := net.Listen("tcp", grpcOptions.Addr)
	if err != nil {
		slog.Error("failed to listen", "error", err)
		return nil, err
	}

	serverOptions = appendTLSCreds(serverOptions, tlsOptions)

	registerFn, _ := registerBuilder()

	sdkContext, err := polaris.NewSDKContextByAddress(polarisOptions.Addr)
	if err != nil {
		return nil, err
	}
	srv, err := polarisgrpc.NewServer(
		polarisgrpc.WithSDKContext(sdkContext),
		polarisgrpc.WithGRPCServerOptions(serverOptions...),
		polarisgrpc.WithServerNamespace(polarisOptions.Provider.Namespace),
		polarisgrpc.WithServiceName(polarisOptions.Provider.Service),
		polarisgrpc.WithServerHost(polarisOptions.Provider.Host),
		// polarisgrpc.WithPort(polarisOptions.Provider.Port),
		polarisgrpc.WithToken(polarisOptions.Provider.Token),
		polarisgrpc.WithServerVersion(polarisOptions.Provider.Version),
		polarisgrpc.WithTTL(polarisOptions.Provider.TTL),
		polarisgrpc.WithHeartbeatEnable(polarisOptions.Provider.Heartbeat),
		// polarisgrpc.WithDelayRegisterEnable(&polarisgrpc.WaitDelayStrategy{WaitTime: 10 * time.Second}),
		polarisgrpc.WithGracefulStopEnable(10*time.Second),
	)
	if err != nil {
		return nil, err
	}

	registerFn(srv.Server)
	healthServer := registerHealthServer(polarisOptions.Provider.Service, srv.Server)
	reflection.Register(srv.Server)

	return &PolarisServer{srv: srv, lis: lis, healthServer: healthServer}, nil
}

// Run 启动 GRPC 服务器并阻塞直到服务器停止或出错.
func (s *PolarisServer) Run(ctx context.Context) error {
	slog.Info("start to listening the incoming requests", "protocol", "grpc", "addr", s.lis.Addr().String())
	if err := s.srv.Serve(s.lis); err != nil {
		slog.Error("failed to serve grpc server", "error", err)
		return err
	}
	return nil
}

// GracefulStop 优雅地关闭 GRPC 服务器.
func (s *PolarisServer) GracefulStop(ctx context.Context) error {
	slog.Info("gracefully stop grpc server")

	// 先将服务置为 NOT_SERVING，使负载均衡/探活停止转发新流量.
	if s.healthServer != nil {
		s.healthServer.Shutdown()
	}

	s.srv.Deregister()
	s.srv.GracefulStop()
	return nil
}
