package watch

import (
	"errors"
	"time"

	"github.com/spf13/pflag"
)

// Options structure holds the configuration options required to create and run a watch server.
type Options struct {
	// LockName specifies the name of the lock used by the server.
	LockName string `json:"lock-name" mapstructure:"lock-name"`

	// HealthzPort is the port number for the health check endpoint.
	HealthzPort int `json:"healthz-port" mapstructure:"healthz-port"`

	// MetricsAddr specifies the address (host:port) for the metrics server endpoint.
	MetricsAddr string `json:"metrics-addr" mapstructure:"metrics-addr"`

	// DisableWatchers is a slice of watchers that will be disabled when the server is run.
	DisableWatchers []string `json:"disable-watchers" mapstructure:"disable-watchers"`

	// MaxWorkers defines the maximum number of concurrent workers that each watcher can spawn.
	MaxWorkers int64 `json:"max-workers" mapstructure:"max-workers"`

	// WatchTimeout defines the timeout duration for each individual watch execution.
	WatchTimeout time.Duration `json:"watch-timeout" mapstructure:"watch-timeout"`

	// PerConcurrency defines the maximum number of concurrent executions allowed for each individual watcher.
	PerConcurrency int `json:"per-watch-concurrency" mapstructure:"per-watch-concurrency"`
}

// NewOptions initializes and returns a new Options instance with default values.
func NewOptions() *Options {
	o := &Options{
		LockName:        "default-distributed-watch-lock",
		HealthzPort:     8881,
		MetricsAddr:     ":9090",
		DisableWatchers: []string{},
		MaxWorkers:      1000,
		WatchTimeout:    30 * time.Second,
		PerConcurrency:  10,
	}

	return o
}

// AddFlags adds the command-line flags associated with the Options structure to the provided FlagSet.
// This will allow users to configure the watch server via command-line arguments.
func (o *Options) AddFlags(fs *pflag.FlagSet, fullPrefix string) {
	fs.StringVar(&o.LockName, fullPrefix+".lock-name", o.LockName,
		"The name of the lock used by the server.")

	fs.IntVar(&o.HealthzPort, fullPrefix+".healthz-port", o.HealthzPort,
		"The port number for the health check endpoint.")

	fs.StringVar(&o.MetricsAddr, fullPrefix+".metrics-addr", o.MetricsAddr,
		"The address (host:port) for the metrics server endpoint.")

	fs.StringSliceVar(&o.DisableWatchers, fullPrefix+".disable-watchers", o.DisableWatchers,
		"The list of watchers that should be disabled.")

	fs.Int64Var(&o.MaxWorkers, fullPrefix+".max-workers", o.MaxWorkers,
		"Specify the maximum concurrency worker of each watcher.")

	fs.DurationVar(&o.WatchTimeout, fullPrefix+".timeout", o.WatchTimeout,
		"The timeout duration for each individual watch execution (e.g., 30s, 2m, 1h).")

	fs.IntVar(&o.PerConcurrency, fullPrefix+".per-concurrency", o.PerConcurrency,
		"The maximum number of concurrent executions allowed for each individual watcher.")
}

// Validate checks the Options structure for required configurations and returns a slice of errors.
func (o *Options) Validate() []error {
	errs := []error{}

	// Validate LockName
	if o.LockName == "" {
		errs = append(errs, errors.New("lock-name cannot be empty"))
	}

	// Validate HealthzPort
	if o.HealthzPort < 0 || o.HealthzPort > 65535 {
		errs = append(errs, errors.New("healthz-port must be between 0 and 65535"))
	}

	// Validate MaxWorkers
	if o.MaxWorkers <= 0 {
		errs = append(errs, errors.New("max-workers must be greater than 0"))
	}

	// Validate WatchTimeout
	if o.WatchTimeout <= 0 {
		errs = append(errs, errors.New("watch-timeout must be greater than 0"))
	}

	// Check for reasonable timeout bounds (optional but recommended)
	if o.WatchTimeout > 24*time.Hour {
		errs = append(errs, errors.New("watch-timeout should not exceed 24 hours for practical reasons"))
	}

	// Validate PerConcurrency
	if o.PerConcurrency <= 0 {
		errs = append(errs, errors.New("per-watch-concurrency must be greater than 0"))
	}

	// Check for reasonable concurrency bounds
	if o.PerConcurrency > 1000 {
		errs = append(errs, errors.New("per-watch-concurrency should not exceed 1000 for resource management"))
	}

	// Cross-validation: PerConcurrency should not exceed MaxWorkers
	if int64(o.PerConcurrency) > o.MaxWorkers {
		errs = append(errs, errors.New("per-watch-concurrency cannot be greater than max-workers"))
	}

	// Validate DisableWatchers (optional: check for duplicates)
	watcherSet := make(map[string]bool)
	for _, watcher := range o.DisableWatchers {
		if watcher == "" {
			errs = append(errs, errors.New("disable-watchers cannot contain empty strings"))
			continue
		}
		if watcherSet[watcher] {
			errs = append(errs, errors.New("disable-watchers contains duplicate entries: "+watcher))
		}
		watcherSet[watcher] = true
	}

	return errs
}
