// Copyright 2024 Coling Kong <colin.kong@bitget.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.

package options

import "github.com/spf13/pflag"

// ServerOptions aggregates the core configuration of a typical server-side
// application. It is the composition entry point for the leaf options in this
// package and is designed to be passed directly to app.WithOptions.
//
// It satisfies the shape required by pkg/app.FlagSetOptions:
//
//	AddFlags(*pflag.FlagSet) and Validate() []error
//
// so the framework can register its flags and validate them before the RunFunc.
type ServerOptions struct {
	// SlogOptions configures structured logging (slog).
	SlogOptions *SlogOptions
	// InsecureServing configures the plaintext HTTP listener.
	InsecureServing *InsecureServingOptions
	// SecureServing configures the HTTPS/TLS listener.
	SecureServing *SecureServingOptions
	// GRPC configures the gRPC listener.
	GRPC *GRPCOptions
	// Health configures the health check server.
	Health *HealthOptions
}

// NewServerOptions creates a ServerOptions instance with default values for all
// embedded leaf options.
func NewServerOptions() *ServerOptions {
	return &ServerOptions{
		SlogOptions:     NewSlogOptions(),
		InsecureServing: NewInsecureServingOptions(),
		SecureServing:   NewSecureServingOptions(),
		GRPC:            NewGRPCOptions(),
		Health:          NewHealthOptions(),
	}
}

// AddFlags registers the flags of every embedded leaf option on fs, using a
// fixed prefix per section (log, insecure, secure, grpc, health).
func (o *ServerOptions) AddFlags(fs *pflag.FlagSet) {
	o.SlogOptions.AddFlags(fs, "log")
	o.InsecureServing.AddFlags(fs, "insecure")
	o.SecureServing.AddFlags(fs, "secure")
	o.GRPC.AddFlags(fs, "grpc")
	o.Health.AddFlags(fs, "health")
}

// Validate aggregates validation of every embedded leaf option.
func (o *ServerOptions) Validate() []error {
	var errs []error
	for _, opt := range []IOptions{o.SlogOptions, o.InsecureServing, o.SecureServing, o.GRPC, o.Health} {
		if opt != nil {
			errs = append(errs, opt.Validate()...)
		}
	}
	return errs
}

// HealthOptions exposes the embedded health check configuration. It lets the
// app framework wire the built-in health check server to these options without
// importing the options package internals (duck-typed, no import cycle).
func (o *ServerOptions) HealthOptions() *HealthOptions {
	return o.Health
}

// Compile-time assertion that ServerOptions satisfies the flag/validate shape
// expected by pkg/app.FlagSetOptions (duck-typed, no import cycle).
var _ interface {
	AddFlags(*pflag.FlagSet)
	Validate() []error
} = (*ServerOptions)(nil)
