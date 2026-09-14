// Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.
//

package options

import (
	"fmt"

	consul "github.com/go-kratos/kratos/contrib/registry/consul/v2"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/spf13/pflag"

	iputil "github.com/onexstack/onexstack/pkg/util/ip"
)

var _ IOptions = (*ConsulOptions)(nil)

// ConsulOptions defines options for consul client.
type ConsulOptions struct {
	// Address is the address of the Consul server
	Addr string `json:"addr,omitempty" mapstructure:"addr"`

	// Scheme is the URI scheme for the Consul server
	Scheme string `json:"scheme,omitempty" mapstructure:"scheme"`

	// ServiceName is the service name registered to consul.
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

// NewConsulOptions create a `zero` value instance.
func NewConsulOptions() *ConsulOptions {
	return &ConsulOptions{
		Addr:           "127.0.0.1:8500",
		Scheme:         "http",
		Host:           iputil.GetLocalIP(),
		EndpointScheme: "http",
	}
}

// Validate verifies flags passed to ConsulOptions.
func (o *ConsulOptions) Validate() []error {
	errs := []error{}

	if o.Addr == "" {
		errs = append(errs, fmt.Errorf("--consul.addr can not be empty"))
	}

	return errs
}

// AddFlags adds flags related to mysql storage for a specific APIServer to the specified FlagSet.
func (o *ConsulOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.Addr, fullPrefix+".addr", o.Addr, ""+
		"Addr is the address of the consul server.")

	fs.StringVar(&o.Scheme, fullPrefix+".scheme", o.Scheme, ""+
		"Scheme is the URI scheme for the consul server.")

	fs.StringVar(&o.ServiceName, fullPrefix+".service-name", o.ServiceName, "Service name registered to consul.")
	fs.StringVar(&o.Host, fullPrefix+".host", o.Host, "Host of the registered service instance.")
	fs.IntVar(&o.Port, fullPrefix+".port", o.Port, "Port of the registered service instance.")
	fs.StringVar(&o.EndpointScheme, fullPrefix+".endpoint-scheme", o.EndpointScheme, "URI scheme of the registered endpoint (http/grpc).")
}

// Register registers the current service instance with consul.
func (o *ConsulOptions) Register() error {
	c := consulapi.DefaultConfig()
	c.Address = o.Addr
	c.Scheme = o.Scheme
	cli, err := consulapi.NewClient(c)
	if err != nil {
		return err
	}

	o.registrar = &registrar{
		r:   consul.New(cli, consul.WithHealthCheck(false)),
		svc: serviceInstance(o.ServiceName, o.EndpointScheme, o.Host, o.Port),
	}
	return o.registrar.Register()
}

// Deregister deregisters the current service instance from consul.
func (o *ConsulOptions) Deregister() error {
	if o.registrar == nil {
		return fmt.Errorf("the service has not been registered, so it cannot be deregistered")
	}
	return o.registrar.Deregister()
}
