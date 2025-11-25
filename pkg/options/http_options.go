// Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.
//

package options

import (
	"time"

	"github.com/spf13/pflag"
)

var _ IOptions = (*HTTPOptions)(nil)

// HTTPOptions contains configuration items related to HTTP server startup.
type HTTPOptions struct {
	// Network with server network.
	Network string `json:"network" mapstructure:"network"`

	// Address with server address.
	Addr string `json:"addr" mapstructure:"addr"`

	// Timeout with server timeout. Used by http client side.
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`
}

// NewHTTPOptions creates a HTTPOptions object with default parameters.
func NewHTTPOptions() *HTTPOptions {
	return &HTTPOptions{
		Network: "tcp",
		Addr:    "0.0.0.0:38443",
		Timeout: 30 * time.Second,
	}
}

// Validate is used to parse and validate the parameters entered by the user at
// the command line when the program starts.
func (o *HTTPOptions) Validate() []error {
	if o == nil {
		return nil
	}

	errors := []error{}

	if err := ValidateAddress(o.Addr); err != nil {
		errors = append(errors, err)
	}

	return errors
}

// AddFlagsWithPrefix registers HTTP server related flags to the specified FlagSet,
// using fullPrefix as the complete prefix for flag names.
//
// Example:
//
//	o.AddFlagsWithPrefix(fs, "apiserver.http")  // --apiserver.http.network, --apiserver.http.addr, etc.
//	o.AddFlagsWithPrefix(fs, "gateway.http")    // --gateway.http.network, --gateway.http.addr, etc.
func (o *HTTPOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.Network, fullPrefix+".network", o.Network,
		"Network type for the HTTP server (e.g., tcp, tcp4, tcp6).")
	fs.StringVar(&o.Addr, fullPrefix+".addr", o.Addr,
		"Listen address for the HTTP server (e.g., :8080, 0.0.0.0:8443).")
	fs.DurationVar(&o.Timeout, fullPrefix+".timeout", o.Timeout,
		"Timeout for incoming HTTP connections.")
}

// Complete fills in any fields not set that are required to have valid data.
func (s *HTTPOptions) Complete() error {
	return nil
}
