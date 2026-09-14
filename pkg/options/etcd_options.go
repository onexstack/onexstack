// Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.
//

package options

import (
	"fmt"
	"time"

	etcd "github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/spf13/pflag"
	clientv3 "go.etcd.io/etcd/client/v3"

	iputil "github.com/onexstack/onexstack/pkg/util/ip"
)

var _ IOptions = (*EtcdOptions)(nil)

// EtcdOptions defines options for etcd cluster.
type EtcdOptions struct {
	Endpoints   []string      `json:"endpoints"               mapstructure:"endpoints"`
	DialTimeout time.Duration `json:"dial-timeout"         mapstructure:"dial-timeout"`
	Username    string        `json:"username"                mapstructure:"username"`
	Password    string        `json:"password"                mapstructure:"password"`
	TLSOptions  TLSOptions    `json:"tls"               mapstructure:"tls"`

	// ServiceName is the service name registered to etcd.
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

// NewEtcdOptions create a `zero` value instance.
func NewEtcdOptions() *EtcdOptions {
	return &EtcdOptions{
		Endpoints:      []string{"127.0.0.1:2379"},
		DialTimeout:    5 * time.Second,
		TLSOptions:     NewTLSOptions(),
		Host:           iputil.GetLocalIP(),
		EndpointScheme: "http",
	}
}

// Validate verifies flags passed to EtcdOptions.
func (o *EtcdOptions) Validate() []error {
	errs := []error{}

	if len(o.Endpoints) == 0 {
		errs = append(errs, fmt.Errorf("--etcd.endpoints can not be empty"))
	}

	if o.DialTimeout <= 0 {
		errs = append(errs, fmt.Errorf("--etcd.dial-timeout cannot be negative"))
	}

	errs = append(errs, o.TLSOptions.Validate()...)

	return errs
}

// AddFlags adds flags related to redis storage for a specific APIServer to the specified FlagSet.
func (o *EtcdOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	o.TLSOptions.AddFlags(fs, fullPrefix+".tls")

	fs.StringSliceVar(&o.Endpoints, fullPrefix+".endpoints", o.Endpoints, "Endpoints of etcd cluster.")
	fs.StringVar(&o.Username, fullPrefix+".username", o.Username, "Username of etcd cluster.")
	fs.StringVar(&o.Password, fullPrefix+".password", o.Password, "Password of etcd cluster.")
	fs.DurationVar(&o.DialTimeout, fullPrefix+".dial-timeout", o.DialTimeout, "Etcd dial timeout in seconds.")

	fs.StringVar(&o.ServiceName, fullPrefix+".service-name", o.ServiceName, "Service name registered to etcd.")
	fs.StringVar(&o.Host, fullPrefix+".host", o.Host, "Host of the registered service instance.")
	fs.IntVar(&o.Port, fullPrefix+".port", o.Port, "Port of the registered service instance.")
	fs.StringVar(&o.EndpointScheme, fullPrefix+".endpoint-scheme", o.EndpointScheme, "URI scheme of the registered endpoint (http/grpc).")
}

// NewClient creates a new etcd client based on the provided options.
func (o *EtcdOptions) NewClient() (*clientv3.Client, error) {
	return clientv3.New(clientv3.Config{
		Endpoints:   o.Endpoints,
		DialTimeout: o.DialTimeout,
		Username:    o.Username,
		Password:    o.Password,
		TLS:         o.TLSOptions.MustTLSConfig(),
	})
}

// Register registers the current service instance with etcd.
func (o *EtcdOptions) Register() error {
	client, err := o.NewClient()
	if err != nil {
		return err
	}

	o.registrar = &registrar{
		r:   etcd.New(client),
		svc: serviceInstance(o.ServiceName, o.EndpointScheme, o.Host, o.Port),
	}
	return o.registrar.Register()
}

// Deregister deregisters the current service instance from etcd.
func (o *EtcdOptions) Deregister() error {
	if o.registrar == nil {
		return fmt.Errorf("the service has not been registered, so it cannot be deregistered")
	}
	return o.registrar.Deregister()
}
