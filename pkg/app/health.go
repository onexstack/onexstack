// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

package app

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"time"
)

// defaultHealthCheckReadHeaderTimeout bounds how long the server waits to read
// request headers, mitigating slowloris-style connection exhaustion.
const defaultHealthCheckReadHeaderTimeout = 5 * time.Second

// Shutdowner is implemented by components that need graceful shutdown at the
// end of the application lifecycle. It is invoked after the PreShutdown hooks,
// under a timeout context (see WithShutdownTimeout).
type Shutdowner interface {
	Shutdown(ctx context.Context) error
}

// healthServer is a minimal, lifecycle-aware health check server. Unlike the
// legacy implementation, it never calls os.Exit: startup errors are returned
// to the caller and runtime serve errors are logged, so shutdown remains
// cooperative and testable.
type healthServer struct {
	server *http.Server
}

// newHealthServer builds a health check server bound to addr serving a liveness
// handler at path. When enableProfiler is true, the standard net/http/pprof
// endpoints are also mounted under /debug/pprof.
func newHealthServer(addr, path string, enableProfiler bool) *healthServer {
	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok"}`))
	})

	if enableProfiler {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}

	return &healthServer{
		server: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: defaultHealthCheckReadHeaderTimeout,
		},
	}
}

// Start binds the listening socket synchronously (so port conflicts are
// reported immediately) and serves in a background goroutine.
func (h *healthServer) Start() error {
	ln, err := net.Listen("tcp", h.server.Addr)
	if err != nil {
		return err
	}

	slog.Info("starting health check server", "addr", h.server.Addr)
	go func() {
		if err := h.server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("health check server stopped unexpectedly", "err", err)
		}
	}()
	return nil
}

// Shutdown gracefully stops the health check server.
func (h *healthServer) Shutdown(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}
