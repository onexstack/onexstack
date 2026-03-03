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

var _ IOptions = (*InsecureServingOptions)(nil)

// InsecureServingOptions contains configuration items related to HTTP server startup.
type InsecureServingOptions struct {
	// Address with server address.
	Addr string `json:"addr" mapstructure:"addr"`

	// Timeout with server timeout. Used by http client side.
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`
}

// NewInsecureServingOptions creates a InsecureServingOptions object with default parameters.
func NewInsecureServingOptions() *InsecureServingOptions {
	return &InsecureServingOptions{
		Addr:    ":8080",
		Timeout: 30 * time.Second,
	}
}

// Validate is used to parse and validate the parameters entered by the user at
// the command line when the program starts.
func (o *InsecureServingOptions) Validate() []error {
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
func (o *InsecureServingOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.Addr, fullPrefix+".addr", o.Addr,
		"Listen address for the HTTP server (e.g., :8080, 0.0.0.0:8443).")
	fs.DurationVar(&o.Timeout, fullPrefix+".timeout", o.Timeout,
		"Timeout for incoming HTTP connections.")
}

// Complete fills in any fields not set that are required to have valid data.
func (s *InsecureServingOptions) Complete() error {
	return nil
}
