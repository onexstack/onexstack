// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

package server

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
)

// 编译期断言：所有 server 实现统一的 Server 接口（Run(ctx) error + GracefulStop(ctx) error）。
var (
	_ Server = (*GRPCServer)(nil)
	_ Server = (*HTTPServer)(nil)
	_ Server = (*GenericAPIServer)(nil)
	_ Server = (*KratosServer)(nil)
	_ Server = (*PolarisServer)(nil)
	_ Server = (*GRPCGatewayServer)(nil)
)

// TestGRPCServerGracefulStopReturnsNil 验证 GracefulStop 在正常路径下返回 nil，
// 且不再静默忽略错误（统一接口后签名返回 error）。
func TestGRPCServerGracefulStopReturnsNil(t *testing.T) {
	s := &GRPCServer{srv: grpc.NewServer()}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := s.GracefulStop(ctx); err != nil {
		t.Fatalf("GracefulStop returned unexpected error: %v", err)
	}
}
