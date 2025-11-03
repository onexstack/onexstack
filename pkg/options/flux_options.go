package options

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"

	"github.com/onexstack/onexstack/pkg/flux"
)

var _ IOptions = (*FluxOptions)(nil)

type Key flux.Key

const (
	RefreshModeSingle string = "single"
	RefreshModeBatch  string = "batch"
	RefreshModeAll    string = "all"
	RefreshModeAuto   string = "auto"
)

// FluxOptions contains configuration items related to Flux cache.
type FluxOptions struct {
	// Cache settings
	MaxKeys     int64         `json:"max-keys,omitempty" mapstructure:"max-keys"`
	TTL         time.Duration `json:"ttl,omitempty" mapstructure:"ttl"`
	BufferItems int64         `json:"buffer-items,omitempty" mapstructure:"buffer-items"`

	// Loader settings
	LoadTimeout time.Duration `json:"load-timeout,omitempty" mapstructure:"load-timeout"`

	// Async refresh settings
	EnableAsync     bool          `json:"enable-async,omitempty" mapstructure:"enable-async"`
	RefreshInterval time.Duration `json:"refresh-interval,omitempty" mapstructure:"refresh-interval"`
	RefreshMode     string        `json:"refresh-mode,omitempty" mapstructure:"refresh-mode"`
	MaxConcurrency  int           `json:"max-concurrency,omitempty" mapstructure:"max-concurrency"`
	BatchSize       int           `json:"batch-size,omitempty" mapstructure:"batch-size"`

	// Monitoring
	EnableStats bool `json:"enable-stats,omitempty" mapstructure:"enable-stats"`
}

// NewFluxOptions creates an Options object with default parameters.
func NewFluxOptions() *FluxOptions {
	return &FluxOptions{
		// Cache settings
		MaxKeys:     1000,
		TTL:         30 * time.Minute,
		BufferItems: 64,

		// Loader settings
		LoadTimeout: 30 * time.Second,

		// Async refresh settings
		EnableAsync:     false,
		RefreshInterval: 2 * time.Minute,
		RefreshMode:     RefreshModeAuto,
		MaxConcurrency:  10,
		BatchSize:       100,

		// Monitoring
		EnableStats: false,
	}
}

// Validate verifies flags passed to FluxOptions.
func (o *FluxOptions) Validate() []error {
	var errs []error

	// Validate MaxKeys
	if o.MaxKeys <= 0 {
		errs = append(errs, fmt.Errorf("max-keys must be greater than 0, got: %d", o.MaxKeys))
	}

	// Validate BufferItems
	if o.BufferItems <= 0 {
		errs = append(errs, fmt.Errorf("buffer-items must be greater than 0, got: %d", o.BufferItems))
	}

	// Validate LoadTimeout
	if o.LoadTimeout <= 0 {
		errs = append(errs, fmt.Errorf("load-timeout must be greater than 0, got: %v", o.LoadTimeout))
	}

	// Validate RefreshInterval (only if async is enabled)
	if o.EnableAsync && o.RefreshInterval <= 0 {
		errs = append(errs, fmt.Errorf("refresh-interval must be greater than 0 when async is enabled, got: %v", o.RefreshInterval))
	}

	// Validate RefreshMode
	switch o.RefreshMode {
	case RefreshModeSingle, RefreshModeBatch, RefreshModeAll, RefreshModeAuto:
	default:
		errs = append(errs, fmt.Errorf("invalid refresh mode: %s (must be single, batch, all or auto)", o.RefreshMode))
	}

	// Validate MaxConcurrency
	if o.EnableAsync && o.MaxConcurrency <= 0 {
		errs = append(errs, fmt.Errorf("max-concurrency must be greater than 0 when async is enabled, got: %d", o.MaxConcurrency))
	}

	// Validate BatchSize
	if o.EnableAsync && o.RefreshMode == RefreshModeBatch && o.BatchSize <= 0 {
		errs = append(errs, fmt.Errorf("batch-size must be greater than 0 when refresh mode is batch, got: %d", o.BatchSize))
	}

	// Validate TTL (allow 0 for no expiration)
	if o.TTL < 0 {
		errs = append(errs, fmt.Errorf("ttl must be non-negative, got: %v", o.TTL))
	}

	return errs
}

// AddFlags adds command line flags for the configuration.
func (o *FluxOptions) AddFlags(fs *pflag.FlagSet, prefixes ...string) {
	prefix := join(prefixes...) + "flux."

	// Cache settings
	fs.Int64Var(&o.MaxKeys, prefix+"max-keys", o.MaxKeys,
		"Maximum number of keys to store in cache.")
	fs.DurationVar(&o.TTL, prefix+"ttl", o.TTL,
		"Time to live for cache entries. Set to 0 for no expiration.")
	fs.Int64Var(&o.BufferItems, prefix+"buffer-items", o.BufferItems,
		"Buffer size for internal operations.")

	// Loader settings
	fs.DurationVar(&o.LoadTimeout, prefix+"load-timeout", o.LoadTimeout,
		"Timeout for data loading operations.")

	// Async refresh settings
	fs.BoolVar(&o.EnableAsync, prefix+"enable-async", o.EnableAsync,
		"Enable asynchronous refresh of cache entries.")
	fs.DurationVar(&o.RefreshInterval, prefix+"refresh-interval", o.RefreshInterval,
		"Interval between async refresh operations.")
	fs.StringVar(&o.RefreshMode, prefix+"refresh-mode", o.RefreshMode,
		"Refresh mode for async operations. Permitted modes: single, batch, auto.")
	fs.IntVar(&o.MaxConcurrency, prefix+"max-concurrency", o.MaxConcurrency,
		"Maximum number of concurrent refresh operations.")
	fs.IntVar(&o.BatchSize, prefix+"batch-size", o.BatchSize,
		"Batch size for batch refresh mode.")

	// Monitoring
	fs.BoolVar(&o.EnableStats, prefix+"enable-stats", o.EnableStats,
		"Enable statistics collection for cache operations.")
}

// ToRefreshMode converts string refresh mode to RefreshMode type.
func (o *FluxOptions) ToRefreshMode() flux.RefreshMode {
	switch o.RefreshMode {
	case RefreshModeSingle:
		return flux.RefreshModeSingle
	case RefreshModeBatch:
		return flux.RefreshModeBatch
	case RefreshModeAll:
		return flux.RefreshModeBatch
	case RefreshModeAuto:
		return flux.RefreshModeAuto
	default:
		// Default to single if unknown mode
		return flux.RefreshModeAuto
	}
}

// ToConfig converts FluxOptions to Config struct (generic version).
// Note: Loader needs to be set separately as it's not configurable via flags.
func ToConfig[K Key, V any](o *FluxOptions) *flux.Config[K, V] {
	return &flux.Config[K, V]{
		// Cache settings
		MaxKeys:     o.MaxKeys,
		TTL:         o.TTL,
		BufferItems: o.BufferItems,

		// Loader settings (needs to be set separately)
		Loader:      nil, // Must be set by caller
		LoadTimeout: o.LoadTimeout,

		// Async refresh settings
		EnableAsync:     o.EnableAsync,
		RefreshInterval: o.RefreshInterval,
		RefreshMode:     o.ToRefreshMode(),
		MaxConcurrency:  o.MaxConcurrency,
		BatchSize:       o.BatchSize,

		// Monitoring
		EnableStats: o.EnableStats,
	}
}

// Apply applies the configuration to create a new Flux cache instance.
// Note: This method requires the loader to be provided.
func Apply[K Key, V any](o *FluxOptions, loader flux.Loader[K, V]) (*flux.Flux[K, V], error) {
	if errs := o.Validate(); len(errs) > 0 {
		return nil, fmt.Errorf("validation failed: %v", errs)
	}

	config := ToConfig[K, V](o)
	config.Loader = loader

	return flux.NewFlux[K, V](flux.WithConfig(config))
}

// BuildConfig creates a Config struct with validation.
// This is a helper function for users who want to get a validated Config.
func BuildConfig[K Key, V any](o *FluxOptions, loader flux.Loader[K, V]) (*flux.Config[K, V], error) {
	if errs := o.Validate(); len(errs) > 0 {
		return nil, fmt.Errorf("validation failed: %v", errs)
	}

	config := ToConfig[K, V](o)
	config.Loader = loader

	return config, nil
}
