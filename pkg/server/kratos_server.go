// Copyright 2024 孔令飞 <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/miniblog. The professional
// version of this repository is https://github.com/onexstack/onex.

package server

import (
	"context"
	"log/slog"
	"net"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
	"github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-kratos/kratos/contrib/registry/eureka/v2"
	"github.com/go-kratos/kratos/contrib/registry/nacos/v2"
	"github.com/go-kratos/kratos/v2"
	krtlog "github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	krtlogger "github.com/onexstack/onexstack/pkg/logger/klog/kratos"
	clientv3 "go.etcd.io/etcd/client/v3"

	genericoptions "github.com/onexstack/onexstack/pkg/options"
)

// The purpose of defining the AppConfig is to demonstrate the usage of wire.Struct.
type KratosAppConfig struct {
	ID        string
	Name      string
	Version   string
	Metadata  map[string]string
	Registrar registry.Registrar
}

// The purpose of defining the AppConfig is to demonstrate the usage of wire.Struct.
type KratosServer struct {
	kapp *kratos.App
}

func NewKratosServer(cfg KratosAppConfig, servers ...transport.Server) (*KratosServer, error) {
	kapp := kratos.New(
		kratos.ID(cfg.ID+"."+cfg.Name),
		kratos.Name(cfg.Name),
		kratos.Version(cfg.Version),
		kratos.Metadata(cfg.Metadata),
		kratos.Logger(NewKratosLogger(cfg.ID, cfg.Name, cfg.Version)),
		kratos.Registrar(cfg.Registrar),
		kratos.Server(servers...),
	)

	return &KratosServer{kapp: kapp}, nil
}

// Run 启动 Kratos 应用并阻塞直到停止或出错。kratos.App.Run 内部自行处理信号
// 与生命周期，因此 ctx 参数此处暂不直接用于触发停止（保留以符合统一 Server 接口）.
func (s *KratosServer) Run(ctx context.Context) error {
	slog.Info("start to listening the incoming requests", "protocol", "kratos")
	if err := s.kapp.Run(); err != nil {
		slog.Error("failed to serve kratos application", "error", err)
		return err
	}
	return nil
}

// GracefulStop 优雅地关闭 Kratos 应用.
func (s *KratosServer) GracefulStop(ctx context.Context) error {
	slog.Info("gracefully stop kratos application")
	if err := s.kapp.Stop(); err != nil {
		slog.Error("Failed to gracefully shutdown kratos application", "error", err)
		return err
	}
	return nil
}

func NewKratosLogger(id, name, version string) krtlog.Logger {
	return krtlog.With(krtlogger.NewLogger(),
		"ts", krtlog.DefaultTimestamp,
		"caller", krtlog.DefaultCaller,
		"service.id", id,
		"service.name", name,
		"service.version", version,
	)
}

func NewEtcdRegistrar(opts *genericoptions.EtcdOptions) registry.Registrar {
	if opts == nil {
		panic("etcd registrar options must be set.")
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   opts.Endpoints,
		DialTimeout: opts.DialTimeout,
		TLS:         opts.TLSOptions.MustTLSConfig(),
		Username:    opts.Username,
		Password:    opts.Password,
	})
	if err != nil {
		panic(err)
	}
	r := etcd.New(client)
	return r
}

func NewConsulRegistrar(opts *genericoptions.ConsulOptions) registry.Registrar {
	if opts == nil {
		panic("consul registrar options must be set.")
	}

	c := consulapi.DefaultConfig()
	c.Address = opts.Addr
	c.Scheme = opts.Scheme
	cli, err := consulapi.NewClient(c)
	if err != nil {
		panic(err)
	}
	r := consul.New(cli, consul.WithHealthCheck(false))
	return r
}

// NewEurekaRegistrar returns a kratos registry.Registrar backed by eureka.
func NewEurekaRegistrar(opts *genericoptions.EurekaOptions) registry.Registrar {
	if opts == nil {
		panic("eureka registrar options must be set.")
	}

	r, err := eureka.New([]string{opts.Addr}, eureka.WithHeartbeat(opts.HeartbeatInterval))
	if err != nil {
		panic(err)
	}
	return r
}

// NewNacosRegistrar returns a kratos registry.Registrar backed by nacos.
func NewNacosRegistrar(opts *genericoptions.NacosOptions) registry.Registrar {
	if opts == nil {
		panic("nacos registrar options must be set.")
	}

	serverConfigs := make([]constant.ServerConfig, 0, len(opts.Endpoints))
	for _, endpoint := range opts.Endpoints {
		host, portStr, err := net.SplitHostPort(endpoint)
		if err != nil {
			panic(err)
		}
		port, err := strconv.ParseUint(portStr, 10, 64)
		if err != nil {
			panic(err)
		}
		serverConfigs = append(serverConfigs, *constant.NewServerConfig(host, port))
	}

	clientConfig := constant.ClientConfig{
		NamespaceId:         opts.Namespace,
		TimeoutMs:           uint64(opts.Timeout / time.Millisecond),
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
		panic(err)
	}

	return nacos.New(client, nacos.WithGroup(opts.Group), nacos.WithCluster(opts.Cluster))
}
