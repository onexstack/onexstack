// Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.
//

package options

import (
	"fmt"
	"time"

	eureka "github.com/go-kratos/kratos/contrib/registry/eureka/v2"
	"github.com/spf13/pflag"

	iputil "github.com/onexstack/onexstack/pkg/util/ip"
)

var _ IOptions = (*EurekaOptions)(nil)

// EurekaOptions defines options for eureka service registry.
type EurekaOptions struct {
	// Addr is the address of the eureka server.
	Addr string `json:"addr" mapstructure:"addr"`
	// HeartbeatInterval is the interval between heartbeats sent to the server.
	HeartbeatInterval time.Duration `json:"heartbeat-interval" mapstructure:"heartbeat-interval"`
	// App is the application name registered to eureka.
	App string `json:"app" mapstructure:"app"`
	// Instance is the instance id registered to eureka.
	Instance string `json:"instance" mapstructure:"instance"`
	// ServiceName is the service name registered to eureka (overrides App when set).
	ServiceName string `json:"service-name,omitempty" mapstructure:"service-name"`
	// Host is the host of the registered service instance.
	Host string `json:"host,omitempty" mapstructure:"host"`
	// Port is the port of the registered service instance.
	Port int `json:"port,omitempty" mapstructure:"port"`
	// EndpointScheme is the URI scheme used to build the registered endpoint (http/grpc).
	EndpointScheme string `json:"endpoint-scheme,omitempty" mapstructure:"endpoint-scheme"`

	// registrar caches the constructed registrar and instance for Deregister.
	registrar *registrar
}

// NewEurekaOptions create a `zero` value instance.
func NewEurekaOptions() *EurekaOptions {
	return &EurekaOptions{
		Addr:              "http://127.0.0.1:8761/eureka",
		HeartbeatInterval: 30 * time.Second,
		App:               "onex",
		Host:              iputil.GetLocalIP(),
		EndpointScheme:    "http",
	}
}

// Validate verifies flags passed to EurekaOptions.
func (o *EurekaOptions) Validate() []error {
	errs := []error{}

	if o.Addr == "" {
		errs = append(errs, fmt.Errorf("--eureka.addr can not be empty"))
	}

	if o.HeartbeatInterval <= 0 {
		errs = append(errs, fmt.Errorf("--eureka.heartbeat-interval cannot be negative"))
	}

	return errs
}

// AddFlags adds flags related to eureka registry to the specified FlagSet.
func (o *EurekaOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.Addr, fullPrefix+".addr", o.Addr, "Address of the eureka server.")
	fs.DurationVar(&o.HeartbeatInterval, fullPrefix+".heartbeat-interval", o.HeartbeatInterval, "Interval between heartbeats sent to the eureka server.")
	fs.StringVar(&o.App, fullPrefix+".app", o.App, "Application name registered to eureka.")
	fs.StringVar(&o.Instance, fullPrefix+".instance", o.Instance, "Instance id registered to eureka.")
	fs.StringVar(&o.ServiceName, fullPrefix+".service-name", o.ServiceName, "Service name registered to eureka.")
	fs.StringVar(&o.Host, fullPrefix+".host", o.Host, "Host of the registered service instance.")
	fs.IntVar(&o.Port, fullPrefix+".port", o.Port, "Port of the registered service instance.")
	fs.StringVar(&o.EndpointScheme, fullPrefix+".endpoint-scheme", o.EndpointScheme, "URI scheme of the registered endpoint (http/grpc).")
}

// Register registers the current service instance with eureka.
func (o *EurekaOptions) Register() error {
	r, err := eureka.New([]string{o.Addr}, eureka.WithHeartbeat(o.HeartbeatInterval))
	if err != nil {
		return err
	}

	name := o.ServiceName
	if name == "" {
		name = o.App
	}

	o.registrar = &registrar{
		r:   r,
		svc: serviceInstance(name, o.EndpointScheme, o.Host, o.Port),
	}
	return o.registrar.Register()
}

// Deregister deregisters the current service instance from eureka.
func (o *EurekaOptions) Deregister() error {
	if o.registrar == nil {
		return fmt.Errorf("the service has not been registered, so it cannot be deregistered")
	}
	return o.registrar.Deregister()
}
