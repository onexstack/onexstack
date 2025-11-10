package watch

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/onexstack/onexstack/pkg/distlock"
	stringsutil "github.com/onexstack/onexstack/pkg/util/strings"
	"github.com/onexstack/onexstack/pkg/watch/initializer"
	"github.com/onexstack/onexstack/pkg/watch/logger/empty"
	"github.com/onexstack/onexstack/pkg/watch/manager"
	"github.com/onexstack/onexstack/pkg/watch/registry"
)

// Option configures a Watch instance with customizable settings.
type Option func(w *Watch)

// Watch represents a monitoring system that schedules and runs tasks at specified intervals.
type Watch struct {
	// Used to implement dist lock.
	db *gorm.DB
	// Job manager to handle scheduling and execution of jobs.
	jm *manager.JobManager
	// Configuration options that control the watch system's runtime behavior.
	options *Options
	// Logger for logging events and errors.
	logger Logger
	// Distributed lock instance.
	locker distlock.Locker
	// Function for internal initialization of watchers.
	initializer initializer.WatcherInitializer
	// Function for external initialization of watchers.
	externalInitializer initializer.WatcherInitializer
	// autoMigrate indicates whether to automatically run database migrations during watcher initialization/startup.
	autoMigrate bool
}

var (
	// Timeout duration for stopping jobs.
	jobStopTimeout = 3 * time.Minute
	// Default expiration time for locks.
	defaultExpiration = 10 * time.Second
)

// WithOptions returns an Option function that applies the provided Options to the Watch.
func WithOptions(opts *Options) Option {
	return func(w *Watch) {
		if opts != nil {
			w.options = opts
		}
	}
}

// WithInitialize returns an Option function that sets the provided WatcherInitializer
// function to initialize the Watch during its creation.
func WithInitialize(initialize initializer.WatcherInitializer) Option {
	return func(w *Watch) {
		w.externalInitializer = initialize
	}
}

// WithLogger returns an Option function that sets the provided Logger to the Watch for logging purposes.
func WithLogger(logger Logger) Option {
	return func(w *Watch) {
		w.logger = logger
	}
}

// WithAutoMigrate returns an Option that enables or disables automatic database migrations.
func WithAutoMigrate(autoMigrate bool) Option {
	return func(w *Watch) {
		w.autoMigrate = autoMigrate
	}
}

// WithLockName returns an Option function that sets the lock name for the Watch.
func WithLockName(lockName string) Option {
	return func(w *Watch) {
		w.options.LockName = lockName
	}
}

// WithHealthzPort returns an Option function that sets the health check port for the Watch.
func WithHealthzPort(port int) Option {
	return func(w *Watch) {
		w.options.HealthzPort = port
	}
}

// WithMetricsAddr returns an Option function that sets the metrics server address for the Watch.
func WithMetricsAddr(addr string) Option {
	return func(w *Watch) {
		w.options.MetricsAddr = addr
	}
}

// WithDisableWatchers returns an Option function that sets the list of disabled watchers for the Watch.
func WithDisableWatchers(watchers []string) Option {
	return func(w *Watch) {
		w.options.DisableWatchers = watchers
	}
}

// WithMaxWorkers returns an Option function that sets the maximum number of workers for the Watch.
func WithMaxWorkers(maxWorkers int64) Option {
	return func(w *Watch) {
		w.options.MaxWorkers = maxWorkers
	}
}

// WithWatchTimeout returns an Option function that sets the timeout duration for each individual watch execution.
func WithWatchTimeout(timeout time.Duration) Option {
	return func(w *Watch) {
		w.options.WatchTimeout = timeout
	}
}

// WithPerConcurrency returns an Option function that sets the maximum number of concurrent executions allowed for each individual watcher.
func WithPerConcurrency(concurrency int) Option {
	return func(w *Watch) {
		w.options.PerConcurrency = concurrency
	}
}

// NewWatch creates a new Watch monitoring system with the provided options.
func NewWatch(opts *Options, db *gorm.DB, withOptions ...Option) (*Watch, error) {
	logger := empty.NewLogger()

	// Create a new Watch with default settings.
	w := &Watch{
		options: opts,
		logger:  logger,
		db:      db,
	}

	// Apply user-defined options to the Watch.
	for _, opt := range withOptions {
		opt(w)
	}

	runner := cron.New(
		cron.WithSeconds(),
		cron.WithLogger(w.logger),
		cron.WithChain(cron.DelayIfStillRunning(w.logger), cron.Recover(w.logger)),
	)

	// Initialize the job manager and the watcher initializer.
	w.jm = manager.NewJobManager(manager.WithCron(runner))
	w.initializer = initializer.NewInitializer(w.jm, w.options.MaxWorkers, w.options.WatchTimeout, w.options.PerConcurrency)

	if err := w.addWatchers(); err != nil {
		return nil, err
	}

	return w, nil
}

// addWatchers initializes all registered watchers and adds them as Cron jobs.
// It skips the watchers that are specified in the disableWatchers slice.
func (w *Watch) addWatchers() error {
	for jobName, watcher := range registry.ListWatchers() {
		if stringsutil.StringIn(jobName, w.options.DisableWatchers) {
			continue
		}

		w.initializer.Initialize(watcher)
		if w.externalInitializer != nil {
			w.externalInitializer.Initialize(watcher)
		}

		spec := registry.Every3Seconds
		if obj, ok := watcher.(registry.ISpec); ok {
			spec = obj.Spec()
		}

		if _, err := w.jm.Add(jobName, spec, watcher); err != nil {
			w.logger.Error(err, "Failed to add job to the cron", "watcher", jobName)
			return err
		}
	}

	return nil
}

// Start attempts to acquire a distributed lock and starts the Cron job scheduler.
// It retries acquiring the lock until successful.
func (w *Watch) Start(stopCh <-chan struct{}) {
	if w.options.HealthzPort != 0 {
		go w.serveHealthz()
	}

	if w.options.MetricsAddr != "" {
		go w.serveMetrics()
	}

	opts := []distlock.Option{
		distlock.WithLockTimeout(defaultExpiration),
		distlock.WithLockName(w.options.LockName),
		distlock.WithAutoMigrate(w.autoMigrate),
	}
	w.locker, _ = distlock.NewGORMLocker(w.db, opts...)
	ticker := time.NewTicker(defaultExpiration + (5 * time.Second))
	ctx := wait.ContextForChannel(stopCh)
	var err error
	for {
		// Obtain a lock for our given mutex. After this is successful, no one else
		// can obtain the same lock (the same mutex name) until we unlock it.
		if err := w.locker.Lock(ctx); err == nil {
			w.logger.Debug("Successfully acquired lock", "lockName", w.options.LockName)
			break
		}

		w.logger.Debug("Failed to acquire lock.", "lockName", w.options.LockName, "error", err)
		<-ticker.C
	}

	w.jm.Start()

	w.logger.Info("Successfully started watch server")
}

// Stop blocks until all jobs are completed and releases the distributed lock.
func (w *Watch) Stop() {
	ctx := w.jm.Stop()
	select {
	case <-ctx.Done():
	case <-time.After(jobStopTimeout):
		w.logger.Error(errors.New("context was not done immediately"), "timeout", jobStopTimeout.String())
	}

	if err := w.locker.Unlock(ctx); err != nil {
		w.logger.Debug("Failed to release lock", "err", err)
	}

	w.logger.Info("Successfully stopped watch server")
}

func (w *Watch) serveMetrics() {
	r := gin.Default()
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	// You can change this port if needed (e.g. ":9090")
	w.logger.Info("Start metrics server", "addr", w.options.MetricsAddr)
	if err := r.Run(w.options.MetricsAddr); err != nil && err != http.ErrServerClosed {
		w.logger.Error(err, "Failed to start metrics server")
		os.Exit(1)
	}
}

// serveHealthz starts the health check server for the Watch instance.
func (w *Watch) serveHealthz() {
	r := mux.NewRouter()
	r.HandleFunc("/healthz", healthzHandler).Methods(http.MethodGet)

	address := fmt.Sprintf("0.0.0.0:%d", w.options.HealthzPort)

	if err := http.ListenAndServe(address, r); err != nil {
		w.logger.Error(err, "Error serving health check endpoint")
	}

	w.logger.Info("Successfully started health check server", "address", address)
}

// healthzHandler handles the health check requests for the service.
func healthzHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-type", "application/json")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(`{"status": "ok"}`))
}
