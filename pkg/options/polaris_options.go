// Copyright 2022 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file. The original repo for
// this file is https://github.com/onexstack/onex.
//

package options

import (
	"fmt"
	"time"

	iputil "github.com/onexstack/onexstack/pkg/util/ip"
	"github.com/polarismesh/polaris-go"
	"github.com/polarismesh/polaris-go/api"
	"github.com/polarismesh/polaris-go/pkg/model"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
)

type ProviderOptions struct {
	// Required
	Namespace string `json:"namespace" mapstructure:"namespace"`
	Service   string `json:"service" mapstructure:"service"`
	Host      string `json:"host" mapstructure:"host"`
	Port      int    `json:"port" mapstructure:"port"`

	// Optional (nil means not set -> let server decide)
	Token string `json:"token" mapstructure:"token"`
	// Protocol      *string           `json:"protocol" mapstructure:"protocol"`
	// Weight        *int              `json:"weight" mapstructure:"weight"`
	// Priority      *int              `json:"priority" mapstructure:"priority"`
	Version string `json:"version" mapstructure:"version"`
	// Metadata      map[string]string `json:"metadata" mapstructure:"metadata"`
	Healthy bool `json:"healthy" mapstructure:"healthy"`
	// Isolate       *bool          `json:"isolate" mapstructure:"isolate"`
	TTL       int  `json:"ttl" mapstructure:"ttl"`
	Heartbeat bool `json:"heartbeat" mapstructure:"heartbeat"`
}

// PolarisOptions defines options for Polaris service.
type PolarisOptions struct {
	// Used by provider and consumer
	Addr string `json:"addr" mapstructure:"addr"`

	Timeout    time.Duration `json:"timeout" mapstructure:"timeout"`
	RetryCount int           `json:"retry-count" mapstructure:"retry-count"`

	// Used by provider
	Provider ProviderOptions `json:"provider" mapstructure:"provider"`

	// Generated fields
	instanceID string `json:"-" mapstructure:"-"`
	provider   api.ProviderAPI
}

// NewPolarisOptions create a `zero` value instance.
func NewPolarisOptions() *PolarisOptions {
	return &PolarisOptions{
		Addr:       "127.0.0.1:8091",
		Timeout:    5 * time.Second,
		RetryCount: 3,
		Provider: ProviderOptions{
			Namespace: "default",
			Host:      iputil.GetLocalIP(),
			Version:   "v0.1.0",
			Healthy:   true,
			TTL:       1,
			Heartbeat: true,
		},
	}
}

// Validate verifies flags passed to PolarisOptions.
func (o *PolarisOptions) Validate() []error {
	errs := []error{}
	return errs
}

// AddFlags adds flags related to Polaris service to the specified FlagSet.
func (o *PolarisOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Addr, "polaris.addr", o.Addr, "Address of your Polaris service(ip:port).")
	fs.DurationVar(&o.Timeout, "polaris.timeout", o.Timeout, "Query timeout per request.")
	fs.IntVar(&o.RetryCount, "polaris.retry-count", o.RetryCount, "Number of retries")

	// required
	fs.StringVar(&o.Provider.Service, "polaris.provider.service", o.Provider.Service, "Service name.")
	fs.StringVar(&o.Provider.Namespace, "polaris.provider.namespace", o.Provider.Namespace, "Namespace.")
	fs.StringVar(&o.Provider.Host, "polaris.provider.host", o.Provider.Host, "Service instance host (IPv4/IPv6).")
	fs.IntVar(&o.Provider.Port, "polaris.provider.port", o.Provider.Port, "Service instance port.")

	// optional scalar
	fs.StringVar(&o.Provider.Token, "polaris.provider.token", o.Provider.Token, "Service access token.")
	// fs.StringVar(&o.InstanceID, "polaris.instance-id", o.InstanceID, "Optional: specify instance id.")
	fs.BoolVar(&o.Provider.Heartbeat, "polaris.provider.heartbeat", o.Provider.Heartbeat, "Let SDK handle heartbeat automatically.")

	// optional pointer-like with has-flags
	// fs.BoolVar(&o.HasProtocol, p+".has-protocol", o.HasProtocol, "Optional: set true to apply --"+p+".protocol.")
	// fs.StringVar(&o.Protocol, p+".protocol", o.Protocol, "Optional: protocol; effective only if --"+p+".has-protocol=true.")

	//fs.BoolVar(&o.HasWeight, p+".has-weight", o.HasWeight, "Optional: set true to apply --"+p+".weight.")
	///fs.IntVar(&o.Weight, p+".weight", o.Weight, "Optional: weight [0..10000]; effective only if --"+p+".has-weight=true.")

	// fs.BoolVar(&o.HasPriority, p+".has-priority", o.HasPriority, "Optional: set true to apply --"+p+".priority.")
	// fs.IntVar(&o.Priority, p+".priority", o.Priority, "Optional: priority (smaller is higher); effective only if --"+p+".has-priority=true.")

	// fs.BoolVar(&o.HasVersion, p+".has-version", o.HasVersion, "Optional: set true to apply --"+p+".version.")
	fs.StringVar(&o.Provider.Version, "polaris.provider.version", o.Provider.Version, "Service version.")

	// fs.BoolVar(&o.HasHealthy, p+".has-healthy", o.HasHealthy, "Optional: set true to apply --"+p+".healthy.")
	fs.BoolVar(&o.Provider.Healthy, "polaris.provider.healthy", o.Provider.Healthy, "To show service is healthy or not.")

	// fs.BoolVar(&o.HasIsolate, p+".has-isolate", o.HasIsolate, "Optional: set true to apply --"+p+".isolate.")
	// fs.BoolVar(&o.Isolate, p+".isolate", o.Isolate, "Optional: is instance isolated; effective only if --"+p+".has-isolate=true.")

	// fs.BoolVar(&o.HasTTL, p+".has-ttl", o.HasTTL, "Optional: set true to apply --"+p+".ttl.")
	fs.IntVar(&o.Provider.TTL, "polaris.provider.ttl", o.Provider.TTL,
		"TTL timeout. if node needs to use heartbeat to report, required. If not set,server will throw ErrorCode-400141")

	// fs.BoolVar(&o.HasTimeout, p+".has-timeout", o.HasTimeout, "Optional: set true to apply --"+p+".timeout.")

	// fs.BoolVar(&o.HasRetry, p+".has-retry", o.HasRetry, "Optional: set true to apply --"+p+".retry-count.")

	// metadata and location
	// fs.StringVar(&o.MetadataKV, p+".metadata", o.MetadataKV, "Optional: metadata as 'k1=v1,k2=v2'.")
	// fs.StringVar(&o.LocationRegion, p+".location-region", o.LocationRegion, "Optional: location.region.")
	// fs.StringVar(&o.LocationZone, p+".location-zone", o.LocationZone, "Optional: location.zone.")
	// fs.StringVar(&o.LocationCampus, p+".location-campus", o.LocationCampus, "Optional: location.campus.")
}

func (o *PolarisOptions) RegisterOrDie() {
	if err := o.Register(); err != nil {
		klog.Fatalf("Failed to register polaris service: %v", err)
	}
}

func (o *PolarisOptions) Register() error {
	provider, err := api.NewProviderAPIByAddress(o.Addr)
	if err != nil {
		return err
	}

	// Register
	service, err := provider.Register(
		&api.InstanceRegisterRequest{
			InstanceRegisterRequest: model.InstanceRegisterRequest{
				Service:      o.Provider.Service,
				ServiceToken: o.Provider.Token,
				Namespace:    o.Provider.Namespace,
				Host:         o.Provider.Host,
				Port:         o.Provider.Port,
				// Protocol:     r.opt.Protocol,
				// Weight:       &r.opt.Weight,
				// Priority:     &r.opt.Priority,
				Version: &o.Provider.Version,
				// Metadata:     rmd,
				Healthy: &o.Provider.Healthy,
				// Isolate:      &r.opt.Isolate,
				TTL:        &o.Provider.TTL,
				Timeout:    &o.Timeout,
				RetryCount: &o.RetryCount,
			},
		})
	if err != nil {
		return err
	}

	o.instanceID = service.InstanceID
	o.provider = provider

	if !o.Provider.Heartbeat {
		return nil
	}

	// start heartbeat report
	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(o.Provider.TTL))
		defer ticker.Stop()

		for {
			<-ticker.C

			err = provider.Heartbeat(&api.InstanceHeartbeatRequest{
				InstanceHeartbeatRequest: model.InstanceHeartbeatRequest{
					Service:      o.Provider.Service,
					Namespace:    o.Provider.Namespace,
					Host:         o.Provider.Host,
					Port:         o.Provider.Port,
					ServiceToken: o.Provider.Token,
					// InstanceID:   instanceID,
					// Timeout:      &r.opt.Timeout,
					// RetryCount:   &r.opt.RetryCount,
				},
			})
			if err != nil {
				klog.ErrorS(err, "Failed to send heartbeat request")
				continue
			}
		}
	}()
	return nil
}

func (o *PolarisOptions) Deregister() error {
	if o.instanceID == "" {
		return fmt.Errorf("The service has not been registered, so it cannot be deregistered.")
	}

	req := &api.InstanceDeRegisterRequest{
		InstanceDeRegisterRequest: model.InstanceDeRegisterRequest{
			Service:      o.Provider.Service,
			ServiceToken: o.Provider.Token,
			Namespace:    o.Provider.Namespace,
			Host:         o.Provider.Host,
			Port:         o.Provider.Port,
			// Timeout:      &r.opt.Timeout,
			// RetryCount:   &r.opt.RetryCount,

			InstanceID: o.instanceID,
		},
	}
	if err := o.provider.Deregister(req); err != nil {
		return err
	}

	return nil
}

func (o *PolarisOptions) NewProviderAPI() (polaris.ProviderAPI, error) {
	return polaris.NewProviderAPIByAddress(o.Addr)
}

func (o *PolarisOptions) NewConsumerAPI() (polaris.ConsumerAPI, error) {
	return polaris.NewConsumerAPIByAddress(o.Addr)
}
