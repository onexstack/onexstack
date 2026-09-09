// Package distlock provides an interface for distributed locking mechanisms.
package distlock

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"errors"
	randv2 "math/rand/v2"
	"os"
	"strconv"
	"time"

	"github.com/onexstack/onexstack/pkg/logger"
	"github.com/onexstack/onexstack/pkg/logger/empty"
)

// DefaultLockName is the default name used for the distributed lock.
const DefaultLockName = "onex-distributed-lock"

// Sentinel errors returned by Locker implementations.
var (
	// ErrLockHeld indicates the lock is currently held by another owner.
	ErrLockHeld = errors.New("distlock: lock is held by another owner")

	// ErrLockNotHeld indicates the current owner no longer holds the lock.
	ErrLockNotHeld = errors.New("distlock: lock is not held by the current owner")
)

// Locker is an interface that defines the methods for a distributed lock.
// It provides methods to acquire, release, and renew a lock in a distributed system.
type Locker interface {
	// Lock attempts to acquire the lock.
	Lock(ctx context.Context) error

	// Unlock releases the previously acquired lock.
	Unlock(ctx context.Context) error

	// Renew updates the expiration time of the lock.
	// It should be called periodically to keep the lock active.
	Renew(ctx context.Context) error
}

// DelayFunc returns the delay to wait before the given retry attempt.
// The attempt index is 1-based.
type DelayFunc func(attempt int) time.Duration

// OnExtendFunc is invoked after each successful lock renewal.
// Returning a non-nil error stops the automatic renewal.
type OnExtendFunc func() error

// Options holds the configuration for the distributed lock.
type Options struct {
	lockName    string        // Name of the lock
	lockTimeout time.Duration // Duration before the lock expires
	ownerID     string        // Identifier for the lock owner
	logger      logger.Logger // Logger for logging events
	autoMigrate bool

	tries     int          // Number of acquisition attempts
	delayFunc DelayFunc    // Delay between acquisition attempts
	onExtend  OnExtendFunc // Callback after each successful renewal
}

// Option is a function that modifies Options.
type Option func(o *Options)

// NewOptions initializes Options with default values.
func NewOptions() *Options {
	return &Options{
		lockName:    DefaultLockName,
		lockTimeout: 10 * time.Second, // Default lock timeout
		ownerID:     newOwnerID(),     // Unique per-instance token
		logger:      empty.NewLogger(),
		tries:       1,                           // Single attempt by default
		delayFunc:   defaultDelayFunc,            // Backoff with jitter
		onExtend:    func() error { return nil }, // No-op callback
	}
}

// ApplyOptions applies a series of Option functions to configure Options.
func ApplyOptions(opts ...Option) *Options {
	o := NewOptions() // Create a new Options instance with default values
	for _, opt := range opts {
		opt(o) // Apply each option to the Options instance
	}

	return o // Return the configured Options
}

// WithLockName sets the lock name in Options.
func WithLockName(name string) Option {
	return func(o *Options) {
		o.lockName = name // Set the lock name
	}
}

// WithLockTimeout sets the lock timeout in Options.
func WithLockTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.lockTimeout = timeout // Set the lock timeout
	}
}

// WithOwnerID sets the owner ID in Options.
func WithOwnerID(ownerID string) Option {
	return func(o *Options) {
		o.ownerID = ownerID // Set the owner ID
	}
}

// WithLogger sets the logger in Options.
func WithLogger(logger logger.Logger) Option {
	return func(o *Options) {
		o.logger = logger // Set the logger
	}
}

// WithAutoMigrate sets the autoMigrate in Options.
func WithAutoMigrate(autoMigrate bool) Option {
	return func(o *Options) {
		o.autoMigrate = autoMigrate
	}
}

// WithTries sets the maximum number of lock acquisition attempts.
// The default is 1 (a single, non-blocking attempt).
func WithTries(n int) Option {
	return func(o *Options) {
		o.tries = n
	}
}

// WithRetryDelay sets a fixed delay between lock acquisition attempts.
// It only takes effect when WithTries is set to a value greater than 1.
func WithRetryDelay(delay time.Duration) Option {
	return func(o *Options) {
		o.delayFunc = func(int) time.Duration { return delay }
	}
}

// WithRetryDelayFunc sets a custom delay function between lock acquisition attempts.
func WithRetryDelayFunc(fn DelayFunc) Option {
	return func(o *Options) {
		o.delayFunc = fn
	}
}

// WithOnExtendFunc sets a callback invoked after each successful lock renewal.
// Returning a non-nil error stops the automatic renewal.
func WithOnExtendFunc(fn OnExtendFunc) Option {
	return func(o *Options) {
		o.onExtend = fn
	}
}

// newOwnerID returns a unique, per-instance owner token composed of the hostname,
// process ID and a random hex suffix. This guarantees that concurrent lock holders
// always use distinct tokens, which is required for safe compare-and-release.
func newOwnerID() string {
	host, _ := os.Hostname()
	b := make([]byte, 8)
	if _, err := crand.Read(b); err != nil {
		return host + "-" + strconv.Itoa(os.Getpid()) + "-" + strconv.FormatInt(time.Now().UnixNano(), 16)
	}

	return host + "-" + strconv.Itoa(os.Getpid()) + "-" + hex.EncodeToString(b)
}

// defaultDelayFunc returns an exponentially-backed-off, jittered delay between
// lock acquisition attempts, capped at 512ms.
func defaultDelayFunc(attempt int) time.Duration {
	const maxShift = 9 // 1ms * 2^9 = 512ms
	shift := attempt
	if shift > maxShift {
		shift = maxShift
	}

	backoff := int64(time.Millisecond) << uint(shift)
	return time.Duration(randv2.Int64N(backoff))
}
