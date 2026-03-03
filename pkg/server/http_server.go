package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"

	genericoptions "github.com/onexstack/onexstack/pkg/options"
	"golang.org/x/sync/errgroup"
)

// HTTPServer represents an HTTP server that supports listening on both HTTP and HTTPS simultaneously.
type HTTPServer struct {
	insecureServer *http.Server
	secureServer   *http.Server
}

// NewHTTPServer creates a new instance of HTTPServer.
func NewHTTPServer(
	insecureOptions *genericoptions.InsecureServingOptions,
	secureOptions *genericoptions.SecureServingOptions,
	handler http.Handler,
) *HTTPServer {
	s := &HTTPServer{}

	// 1. Initialize insecure HTTP server (if address is configured)
	if insecureOptions != nil && insecureOptions.Addr != "" {
		s.insecureServer = &http.Server{
			Addr:              insecureOptions.Addr,
			Handler:           handler,
			ReadHeaderTimeout: insecureOptions.Timeout,
			ReadTimeout:       insecureOptions.Timeout,
			WriteTimeout:      insecureOptions.Timeout,
			IdleTimeout:       3 * insecureOptions.Timeout, // Typically IdleTimeout is longer
		}
	}

	// 2. Initialize secure HTTPS server (if configured and enabled)
	if secureOptions != nil && secureOptions.Enabled {
		tlsConfig := secureOptions.MustTLSConfig()
		s.secureServer = &http.Server{
			Addr:              secureOptions.Addr,
			Handler:           handler,
			TLSConfig:         tlsConfig,
			ReadHeaderTimeout: secureOptions.Timeout,
			ReadTimeout:       secureOptions.Timeout,
			WriteTimeout:      secureOptions.Timeout,
			IdleTimeout:       3 * secureOptions.Timeout, // Typically IdleTimeout is longer
		}
	}

	return s
}

// Run starts the servers. Unlike os.Exit, it returns an error to be handled by the caller,
// which is more idiomatic in Go.
func (s *HTTPServer) Run(ctx context.Context) error {
	eg, _ := errgroup.WithContext(ctx)

	// Start HTTP server
	if s.insecureServer != nil {
		slog.Info("starting insecure HTTP server", "addr", s.insecureServer.Addr)
		eg.Go(func() error {
			if err := s.insecureServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Error("failed to start insecure HTTP server", "error", err)
				return err
			}
			return nil
		})
	}

	// Start HTTPS server
	if s.secureServer != nil {
		slog.Info("starting secure HTTPS server", "addr", s.secureServer.Addr)
		eg.Go(func() error {
			// Key/Cert file paths are usually handled when loading TLSConfig, or passed here.
			// Since TLSConfig already contains the certificates (via GetCertificate or Certificates),
			// we pass empty strings here.
			if err := s.secureServer.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
				slog.Error("failed to start secure HTTPS server", "error", err)
				return err
			}
			return nil
		})
	}

	// Block and wait until one of the servers fails or the Context is canceled.
	// If it returns nil, it means a normal shutdown (http.ErrServerClosed was ignored).
	return eg.Wait()
}

// RunOrDie is a wrapper for compatibility that keeps the original signature.
func (s *HTTPServer) RunOrDie(ctx context.Context) {
	if err := s.Run(ctx); err != nil {
		os.Exit(1)
	}
}

// GracefulStop gracefully stops all servers.
func (s *HTTPServer) GracefulStop(ctx context.Context) {
	slog.Info("gracefully stopping HTTP(s) servers")

	// Use errgroup for concurrent shutdown to improve efficiency.
	eg, ctx := errgroup.WithContext(ctx)

	if s.insecureServer != nil {
		eg.Go(func() error {
			return s.insecureServer.Shutdown(ctx)
		})
	}

	if s.secureServer != nil {
		eg.Go(func() error {
			return s.secureServer.Shutdown(ctx)
		})
	}

	// Wait for all shutdowns to complete.
	if err := eg.Wait(); err != nil {
		slog.Error("one or more servers failed to shutdown gracefully", "error", err)
	} else {
		slog.Info("all servers stopped successfully")
	}
}
