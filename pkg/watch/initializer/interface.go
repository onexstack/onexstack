package initializer

import (
	"time"

	"github.com/onexstack/onexstack/pkg/watch/manager"
	"github.com/onexstack/onexstack/pkg/watch/registry"
	"go.uber.org/ratelimit"
)

// WatcherInitializer is used for initialization of shareable resources between watcher plugins.
// After initialization the resources have to be set separately.
type WatcherInitializer interface {
	Initialize(watcher registry.Watcher)
}

// WantsJobManager defines a function which sets job manager for watcher plugins that need it.
type WantsJobManager interface {
	registry.Watcher
	SetJobManager(jm *manager.JobManager)
}

// WantsRateLimit defines an interface for watchers that need to configure rate limiting.
type WantsRateLimit interface {
	registry.Watcher
	// SetRateLimit sets the maximum number of operations allowed per second for this watcher.
	SetRateLimit(rateLimit ratelimit.Limiter)
}

// WantsWatchTimeout defines an interface for watchers that need to configure execution timeout.
type WantsWatchTimeout interface {
	registry.Watcher
	// SetWatchTimeout sets the maximum duration allowed for a single watch execution.
	SetWatchTimeout(timeout time.Duration)
}

// WantsPerConcurrency defines an interface for watchers that need to configure concurrency limits.
type WantsPerConcurrency interface {
	registry.Watcher
	// SetPerConcurrency sets the maximum number of concurrent executions allowed for this watcher.
	SetPerConcurrency(concurrency int)
}
