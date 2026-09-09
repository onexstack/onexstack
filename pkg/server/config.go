// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

package server

import (
	"context"
	"net/http"

	"google.golang.org/grpc"

	genericoptions "github.com/onexstack/onexstack/pkg/options"
)

// Config aggregates the runtime configuration needed to build a
// GenericAPIServer. It is derived from options.ServerOptions and enriched with
// the business-provided HTTP handler and gRPC service registration.
type Config struct {
	InsecureServing *genericoptions.InsecureServingOptions
	SecureServing   *genericoptions.SecureServingOptions
	GRPC            *genericoptions.GRPCOptions

	// Handler is the HTTP request handler (business routes). Set by the caller.
	Handler http.Handler

	// GRPCServerOptions holds additional gRPC server options (e.g. interceptors).
	GRPCServerOptions []grpc.ServerOption
	// RegisterGRPC registers the business gRPC services on the server.
	RegisterGRPC func(grpc.ServiceRegistrar)
	// GRPCServiceName is reported as the serving service in gRPC health checks.
	GRPCServiceName string
}

// NewConfig converts a ServerOptions into a runtime Config. The HTTP handler
// and gRPC registration must be filled in by the caller before New is invoked.
func NewConfig(opts *genericoptions.ServerOptions) *Config {
	return &Config{
		InsecureServing: opts.InsecureServing,
		SecureServing:   opts.SecureServing,
		GRPC:            opts.GRPC,
	}
}

// New builds a GenericAPIServer from the config. It composes an HTTP server
// (insecure and/or secure) and, when a gRPC registrar is provided, a gRPC
// server. ctx is reserved for future use (e.g. pre-start resource init).
func (c *Config) New(ctx context.Context) (*GenericAPIServer, error) {
	var httpServer *HTTPServer
	if c.InsecureServing != nil || c.SecureServing != nil {
		httpServer = NewHTTPServer(c.InsecureServing, c.SecureServing, c.Handler)
	}

	var grpcServer *GRPCServer
	if c.GRPC != nil && c.RegisterGRPC != nil {
		serverName := c.GRPCServiceName
		if serverName == "" {
			serverName = "grpc-server"
		}

		var tlsOpts *genericoptions.TLSOptions
		if c.SecureServing != nil {
			tlsOpts = &c.SecureServing.TLSOptions
		}

		srv, err := NewGRPCServer(c.GRPC, tlsOpts, c.GRPCServerOptions, func() (func(grpc.ServiceRegistrar), string) {
			return c.RegisterGRPC, serverName
		})
		if err != nil {
			return nil, err
		}
		grpcServer = srv
	}

	return NewGenericAPIServer(httpServer, grpcServer), nil
}
