// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

package server

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"golang.org/x/sync/errgroup"
)

// defaultShutdownTimeout is the default duration to wait for servers to
// gracefully stop before giving up.
const defaultShutdownTimeout = 10 * time.Second

// GenericAPIServer composes the HTTP and gRPC servers into a single runnable
// unit. Its Run method is context-aware: it blocks until the context is
// canceled (e.g. on a shutdown signal) or a server fails, then gracefully
// stops all underlying servers.
type GenericAPIServer struct {
	http            *HTTPServer
	grpc            *GRPCServer
	shutdownTimeout time.Duration
}

// NewGenericAPIServer creates a GenericAPIServer composing the given HTTP and
// gRPC servers. Either may be nil to run only one protocol.
func NewGenericAPIServer(httpServer *HTTPServer, grpcServer *GRPCServer) *GenericAPIServer {
	return &GenericAPIServer{
		http:            httpServer,
		grpc:            grpcServer,
		shutdownTimeout: defaultShutdownTimeout,
	}
}

// Run starts all underlying servers and blocks until ctx is canceled or a
// server fails. On cancellation it gracefully stops the servers and returns nil.
func (s *GenericAPIServer) Run(ctx context.Context) error {
	eg, egCtx := errgroup.WithContext(ctx)

	if s.http != nil {
		eg.Go(func() error { return s.http.Run(egCtx) })
	}
	if s.grpc != nil {
		eg.Go(func() error { return s.grpc.Run(egCtx) })
	}

	// Watch for shutdown (signal cancellation or a server error) and gracefully
	// stop all servers.
	eg.Go(func() error {
		<-egCtx.Done()
		slog.Info("shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()
		return s.GracefulStop(shutdownCtx)
	})

	return eg.Wait()
}

// GracefulStop gracefully stops all underlying servers.
func (s *GenericAPIServer) GracefulStop(ctx context.Context) error {
	var errs []error
	if s.http != nil {
		if err := s.http.GracefulStop(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if s.grpc != nil {
		if err := s.grpc.GracefulStop(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
