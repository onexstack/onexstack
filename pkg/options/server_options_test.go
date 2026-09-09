// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

package options

import (
	"testing"

	"github.com/spf13/pflag"
)

func TestNewServerOptions(t *testing.T) {
	opts := NewServerOptions()

	if opts.SlogOptions == nil {
		t.Fatal("SlogOptions is nil")
	}
	if opts.InsecureServing == nil {
		t.Fatal("InsecureServing is nil")
	}
	if opts.SecureServing == nil {
		t.Fatal("SecureServing is nil")
	}
	if opts.GRPC == nil {
		t.Fatal("GRPC is nil")
	}
	if opts.Health == nil {
		t.Fatal("Health is nil")
	}

	if got := opts.InsecureServing.Addr; got != ":8080" {
		t.Errorf("InsecureServing.Addr = %q, want :8080", got)
	}
	if got := opts.SlogOptions.Level; got != "info" {
		t.Errorf("SlogOptions.Level = %q, want info", got)
	}
}

func TestServerOptionsAddFlags(t *testing.T) {
	opts := NewServerOptions()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	opts.AddFlags(fs)

	for _, name := range []string{
		"log.level",
		"insecure.addr",
		"secure.addr",
		"grpc.addr",
		"health.check-address",
	} {
		if fs.Lookup(name) == nil {
			t.Errorf("expected flag %q to be registered", name)
		}
	}
}

func TestServerOptionsValidate(t *testing.T) {
	opts := NewServerOptions()

	if errs := opts.Validate(); len(errs) != 0 {
		t.Fatalf("expected no validation errors for defaults, got %v", errs)
	}

	// An invalid gRPC address must surface as a validation error.
	opts.GRPC.Addr = "not-a-valid-address"
	if errs := opts.Validate(); len(errs) == 0 {
		t.Fatal("expected a validation error for an invalid gRPC address")
	}
}
