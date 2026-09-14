// Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.
//

package options

import (
	"fmt"
	"net"
	"strconv"
	"time"

	nacos "github.com/go-kratos/kratos/contrib/registry/nacos/v2"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/spf13/pflag"

	iputil "github.com/onexstack/onexstack/pkg/util/ip"
)

var _ IOptions = (*NacosOptions)(nil)

// NacosOptions defines options for nacos service registry.
type NacosOptions struct {
	// Endpoints are the nacos server endpoints (host:port).
	Endpoints []string `json:"endpoints" mapstructure:"endpoints"`
	// Namespace is the nacos namespace id.
	Namespace string `json:"namespace" mapstructure:"namespace"`
	// Group is the nacos service group.
	Group string `json:"group" mapstructure:"group"`
	// Cluster is the nacos cluster name.
	Cluster string `json:"cluster" mapstructure:"cluster"`
	// Timeout is the timeout for nacos client requests.
	Timeout time.Duration `json:"timeout" mapstructure:"timeout"`

	// ServiceName is the service name registered to nacos.
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

// NewNacosOptions create a `zero` value instance.
func NewNacosOptions() *NacosOptions {
	return &NacosOptions{
		Endpoints:      []string{"127.0.0.1:8848"},
		Namespace:      "public",
		Group:          "DEFAULT_GROUP",
		Timeout:        5 * time.Second,
		Host:           iputil.GetLocalIP(),
		EndpointScheme: "http",
	}
}

// Validate verifies flags passed to NacosOptions.
func (o *NacosOptions) Validate() []error {
	errs := []error{}

	if len(o.Endpoints) == 0 {
		errs = append(errs, fmt.Errorf("--nacos.endpoints can not be empty"))
	}

	if o.Timeout <= 0 {
		errs = append(errs, fmt.Errorf("--nacos.timeout cannot be negative"))
	}

	return errs
}

// AddFlags adds flags related to nacos registry to the specified FlagSet.
func (o *NacosOptions) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringSliceVar(&o.Endpoints, fullPrefix+".endpoints", o.Endpoints, "Endpoints of the nacos server.")
	fs.StringVar(&o.Namespace, fullPrefix+".namespace", o.Namespace, "Namespace id of the nacos server.")
	fs.StringVar(&o.Group, fullPrefix+".group", o.Group, "Group of the nacos service.")
	fs.StringVar(&o.Cluster, fullPrefix+".cluster", o.Cluster, "Cluster of the nacos service.")
	fs.DurationVar(&o.Timeout, fullPrefix+".timeout", o.Timeout, "Timeout for nacos client requests.")

	fs.StringVar(&o.ServiceName, fullPrefix+".service-name", o.ServiceName, "Service name registered to nacos.")
	fs.StringVar(&o.Host, fullPrefix+".host", o.Host, "Host of the registered service instance.")
	fs.IntVar(&o.Port, fullPrefix+".port", o.Port, "Port of the registered service instance.")
	fs.StringVar(&o.EndpointScheme, fullPrefix+".endpoint-scheme", o.EndpointScheme, "URI scheme of the registered endpoint (http/grpc).")
}

// Register registers the current service instance with nacos.
func (o *NacosOptions) Register() error {
	serverConfigs := make([]constant.ServerConfig, 0, len(o.Endpoints))
	for _, endpoint := range o.Endpoints {
		host, portStr, err := net.SplitHostPort(endpoint)
		if err != nil {
			return err
		}
		port, err := strconv.ParseUint(portStr, 10, 64)
		if err != nil {
			return err
		}
		serverConfigs = append(serverConfigs, *constant.NewServerConfig(host, port))
	}

	clientConfig := constant.ClientConfig{
		NamespaceId:         o.Namespace,
		TimeoutMs:           uint64(o.Timeout / time.Millisecond),
		NotLoadCacheAtStart: true,
		LogDir:              "/tmp/nacos/log",
		CacheDir:            "/tmp/nacos/cache",
		LogLevel:            "warn",
	}

	client, err := clients.NewNamingClient(vo.NacosClientParam{
		ClientConfig:  &clientConfig,
		ServerConfigs: serverConfigs,
	})
	if err != nil {
		return err
	}

	o.registrar = &registrar{
		r:   nacos.New(client, nacos.WithGroup(o.Group), nacos.WithCluster(o.Cluster)),
		svc: serviceInstance(o.ServiceName, o.EndpointScheme, o.Host, o.Port),
	}
	return o.registrar.Register()
}

// Deregister deregisters the current service instance from nacos.
func (o *NacosOptions) Deregister() error {
	if o.registrar == nil {
		return fmt.Errorf("the service has not been registered, so it cannot be deregistered")
	}
	return o.registrar.Deregister()
}
