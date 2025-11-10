package initializer

import (
	"time"

	"github.com/onexstack/onexstack/pkg/watch/manager"
	"github.com/onexstack/onexstack/pkg/watch/registry"
	"go.uber.org/ratelimit"
)

// watcherInitializer is responsible for initializing specific watcher plugins.
type watcherInitializer struct {
	jm *manager.JobManager
	// Specify the maximum concurrency event of user watcher.
	rl             ratelimit.Limiter
	watchTimeout   time.Duration
	perConcurrency int
}

// Ensure that watcherInitializer implements the WatcherInitializer interface.
var _ WatcherInitializer = (*watcherInitializer)(nil)

// NewInitializer creates and returns a new watcherInitializer instance.
func NewInitializer(jm *manager.JobManager, maxWorkers int64, timeout time.Duration, concur int) *watcherInitializer {
	return &watcherInitializer{
		jm:             jm,
		rl:             ratelimit.New(int(maxWorkers)),
		watchTimeout:   timeout,
		perConcurrency: concur,
	}
}

// Initialize configures the provided watcher by setting up the necessary dependencies
// such as the JobManager and maximum workers.
func (i *watcherInitializer) Initialize(wc registry.Watcher) {
	// We can set a specific configuration as needed, as shown in the example below.
	// However, for convenience, I directly assign all configurations to each watcher,
	// allowing the watcher to choose which ones to use.
	if wants, ok := wc.(WantsJobManager); ok {
		wants.SetJobManager(i.jm)
	}

	if wants, ok := wc.(WantsRateLimit); ok {
		wants.SetRateLimit(i.rl)
	}

	if wants, ok := wc.(WantsWatchTimeout); ok {
		wants.SetWatchTimeout(i.watchTimeout)
	}

	if wants, ok := wc.(WantsPerConcurrency); ok {
		wants.SetPerConcurrency(i.perConcurrency)
	}
}
