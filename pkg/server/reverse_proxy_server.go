// Copyright 2024 孔令飞 <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/miniblog. The professional
// version of this repository is https://github.com/onexstack/onex.

package server

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"

	genericoptions "github.com/onexstack/onexstack/pkg/options"
)

// GRPCGatewayServer 代表一个 GRPC 网关服务器.
type GRPCGatewayServer struct {
	srv *http.Server
}

// NewGRPCGatewayServer 创建一个新的 GRPC 网关服务器实例.
func NewGRPCGatewayServer(
	insecureOptions *genericoptions.InsecureServingOptions,
	grpcOptions *genericoptions.GRPCOptions,
	secureOptions *genericoptions.SecureServingOptions,
	registerHandler func(mux *runtime.ServeMux, conn *grpc.ClientConn) error,
) (*GRPCGatewayServer, error) {
	var tlsConfig *tls.Config
	if secureOptions != nil && secureOptions.Enabled {
		tlsConfig = secureOptions.MustTLSConfig()
		tlsConfig.InsecureSkipVerify = true
	}

	dialOptions := []grpc.DialOption{
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff:           backoff.DefaultConfig,
			MinConnectTimeout: 10 * time.Second, // 最小连接超时时间
		}),
	}
	if tlsConfig != nil {
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(grpcOptions.Addr, dialOptions...)
	if err != nil {
		slog.Error("failed to dial context", "error", err)
		return nil, err
	}

	gwmux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			// 设置序列化 protobuf 数据时，枚举类型的字段以数字格式输出.
			// 否则，默认会以字符串格式输出，跟枚举类型定义不一致，带来理解成本.
			UseEnumNumbers: true,
		},
	}))
	if err := registerHandler(gwmux, conn); err != nil {
		slog.Error("failed to register handler", "error", err)
		return nil, err
	}

	return &GRPCGatewayServer{
		srv: &http.Server{
			Addr:      insecureOptions.Addr,
			Handler:   gwmux,
			TLSConfig: tlsConfig,
		},
	}, nil
}

// RunOrDie 启动 GRPC 网关服务器并在出错时记录致命错误.
func (s *GRPCGatewayServer) RunOrDie() {
	slog.Info("start to listening the incoming requests", "protocol", protocolName(s.srv), "addr", s.srv.Addr)
	// 默认启动 HTTP 服务器
	serveFn := func() error { return s.srv.ListenAndServe() }
	if s.srv.TLSConfig != nil {
		serveFn = func() error { return s.srv.ListenAndServeTLS("", "") }
	}

	if err := serveFn(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to server HTTP(s) server", "error", err)
		os.Exit(1)
	}
}

// GracefulStop 优雅地关闭 GRPC 网关服务器.
func (s *GRPCGatewayServer) GracefulStop(ctx context.Context) {
	slog.Info("gracefully stop HTTP(s) server")
	if err := s.srv.Shutdown(ctx); err != nil {
		slog.Error("http(s) server forced to shutdown", "error", err)
	}
}
