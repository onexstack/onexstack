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
	"net/http"

	"k8s.io/klog/v2"

	genericoptions "github.com/onexstack/onexstack/pkg/options"
)

// HTTPServer 代表一个 HTTP 服务器.
type HTTPServer struct {
	srv *http.Server
}

// NewHTTPServer 创建一个新的 HTTP 服务器实例.
func NewHTTPServer(httpOptions *genericoptions.HTTPOptions, tlsOptions *genericoptions.TLSOptions, handler http.Handler) *HTTPServer {
	var tlsConfig *tls.Config
	if tlsOptions != nil && tlsOptions.UseTLS {
		tlsConfig = tlsOptions.MustTLSConfig()
	}

	return &HTTPServer{
		srv: &http.Server{
			Addr:      httpOptions.Addr,
			Handler:   handler,
			TLSConfig: tlsConfig,
		},
	}
}

// RunOrDie 启动 HTTP 服务器并在出错时记录致命错误.
func (s *HTTPServer) RunOrDie() {
	klog.InfoS("Start to listening the incoming requests", "protocol", protocolName(s.srv), "addr", s.srv.Addr)
	// 默认启动 HTTP 服务器
	serveFn := func() error { return s.srv.ListenAndServe() }
	if s.srv.TLSConfig != nil {
		serveFn = func() error { return s.srv.ListenAndServeTLS("", "") }
	}

	if err := serveFn(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		klog.Fatalf("Failed to server HTTP(s) server: %v", err)
	}
}

// GracefulStop 优雅地关闭 HTTP 服务器.
func (s *HTTPServer) GracefulStop(ctx context.Context) {
	klog.InfoS("Gracefully stop HTTP(s) server")
	if err := s.srv.Shutdown(ctx); err != nil {
		klog.ErrorS(err, "HTTP(s) server forced to shutdown")
	}
}
