package watch

import (
	"sync"
	"time"

	"go.uber.org/ratelimit"
)

// Watcher provides a base implementation for watchers with common functionality.
type Watcher struct {
	// mu provides thread-safe access to the watcher's configuration fields.
	// All getter and setter methods should use this mutex to prevent data races
	// when the watcher is accessed from multiple goroutines.
	mu sync.RWMutex

	// WatchTimeout defines the maximum duration allowed for each individual watch execution.
	// If a watch operation exceeds this timeout, it will be cancelled to prevent hanging.
	// Default value should be set by the implementing watcher if this field is zero.
	WatchTimeout time.Duration

	// PerConcurrency defines the maximum number of concurrent executions allowed
	// for each individual watcher instance. This helps control resource usage
	// and prevents system overload when processing multiple items simultaneously.
	// A value of 0 or negative number should use a sensible default (typically 5).
	PerConcurrency int

	// Limiter provides rate limiting functionality to control the frequency of operations.
	// This helps prevent overwhelming downstream services and ensures fair resource usage.
	// Can be nil if no rate limiting is desired.
	Limiter ratelimit.Limiter
}

// SetWatchTimeout implements WantsWatchTimeout interface.
func (w *Watcher) SetWatchTimeout(timeout time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.WatchTimeout = timeout
}

// GetWatchTimeout returns the configured watch timeout duration.
func (w *Watcher) GetWatchTimeout() time.Duration {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.WatchTimeout > 0 {
		return w.WatchTimeout
	}
	return 30 * time.Second // Default timeout
}

// SetPerConcurrency implements WantsPerConcurrency interface.
// It sets the maximum number of concurrent executions allowed for the watcher.
// This method is thread-safe and can be called concurrently.
func (w *Watcher) SetPerConcurrency(concurrency int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.PerConcurrency = concurrency
}

// GetPerConcurrency returns the configured concurrency limit.
// If no concurrency is configured (zero or negative value), it returns a sensible default.
// This method is thread-safe and can be called concurrently.
func (w *Watcher) GetPerConcurrency() int {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.PerConcurrency > 0 {
		return w.PerConcurrency
	}
	return 5 // Default concurrency
}

// SetRateLimit implements WantsRateLimit interface.
// It configures rate limiting for the watcher operations.
// This method is thread-safe and can be called concurrently.
func (w *Watcher) SetRateLimit(limiter ratelimit.Limiter) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.Limiter = limiter
}

// GetRateLimit returns the configured rate limiter.
// This method is thread-safe and can be called concurrently.
func (w *Watcher) GetRateLimit() ratelimit.Limiter {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.Limiter
}
