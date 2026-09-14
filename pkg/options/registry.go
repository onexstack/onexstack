// Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.
//

package options

import (
	"context"
	"fmt"

	kregistry "github.com/go-kratos/kratos/v2/registry"
)

// serviceInstance builds a kratos registry.ServiceInstance from registration fields.
// All kratos contrib registries (consul/etcd/eureka/nacos) consume Endpoints of the
// form "scheme://host:port", so we encode the single registered endpoint here.
func serviceInstance(name, scheme, host string, port int) *kregistry.ServiceInstance {
	return &kregistry.ServiceInstance{
		ID:        fmt.Sprintf("%s-%s-%d", name, host, port),
		Name:      name,
		Endpoints: []string{fmt.Sprintf("%s://%s:%d", scheme, host, port)},
	}
}

// registrar is the shared state used by registry options to register/deregister a
// service instance. It caches the constructed registrar and instance so that
// Deregister reuses the exact same identity, mirroring PolarisOptions.
type registrar struct {
	r   kregistry.Registrar
	svc *kregistry.ServiceInstance
}

// Register registers the cached service instance with the registry.
func (g *registrar) Register() error {
	return g.r.Register(context.Background(), g.svc)
}

// Deregister deregisters the cached service instance from the registry.
func (g *registrar) Deregister() error {
	return g.r.Deregister(context.Background(), g.svc)
}
