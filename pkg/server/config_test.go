// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

package server

import (
	"context"
	"net/http"
	"testing"
	"time"

	genericoptions "github.com/onexstack/onexstack/pkg/options"
)

// TestConfigNewAndRun verifies the full wiring from options.ServerOptions to a
// runnable GenericAPIServer that gracefully stops when the context is canceled.
func TestConfigNewAndRun(t *testing.T) {
	opts := genericoptions.NewServerOptions()
	opts.InsecureServing.Addr = "127.0.0.1:0" // random free port
	opts.SecureServing = nil
	opts.GRPC = nil

	cfg := NewConfig(opts)
	cfg.Handler = http.NewServeMux()

	srv, err := cfg.New(context.Background())
	if err != nil {
		t.Fatalf("Config.New failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- srv.Run(ctx) }()

	// Give the server a moment to start listening.
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned an error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}
