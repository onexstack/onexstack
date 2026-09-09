package distlock

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/onexstack/onexstack/pkg/logger"
)

// unlockScript atomically deletes the lock only if it is still held by the
// current owner, preventing a stale holder from deleting another owner's lock.
var unlockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
else
	return 0
end
`)

// renewScript atomically extends the lock TTL only if it is still held by the
// current owner, preventing a stale holder from extending another owner's lock.
var renewScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("PEXPIRE", KEYS[1], ARGV[2])
else
	return 0
end
`)

// RedisLocker provides a distributed locking mechanism using Redis.
type RedisLocker struct {
	client      *redis.Client
	lockName    string
	lockTimeout time.Duration
	mu          sync.Mutex
	ownerID     string
	logger      logger.Logger

	tries     int
	delayFunc DelayFunc
	onExtend  OnExtendFunc

	cancel context.CancelFunc
}

// Ensure RedisLocker implements the Locker interface.
var _ Locker = (*RedisLocker)(nil)

// NewRedisLocker creates a new RedisLocker instance.
func NewRedisLocker(client *redis.Client, opts ...Option) *RedisLocker {
	o := ApplyOptions(opts...)
	locker := &RedisLocker{
		client:      client,
		lockName:    o.lockName,
		lockTimeout: o.lockTimeout,
		ownerID:     o.ownerID,
		logger:      o.logger,
		tries:       o.tries,
		delayFunc:   o.delayFunc,
		onExtend:    o.onExtend,
	}

	locker.logger.Info("RedisLocker initialized", "lockName", locker.lockName, "ownerID", locker.ownerID)
	return locker
}

// Lock attempts to acquire the distributed lock.
func (l *RedisLocker) Lock(ctx context.Context) error {
	var lastErr error
	for i := 0; i < l.tries; i++ {
		if i > 0 {
			if err := l.waitRetry(ctx, i); err != nil {
				return err
			}
		}

		lastErr = l.acquireOnce(ctx)
		if lastErr == nil {
			return nil
		}
		if !errors.Is(lastErr, ErrLockHeld) {
			// A real error (e.g. Redis unavailable); do not retry.
			return lastErr
		}
	}

	return lastErr
}

// acquireOnce makes a single attempt to acquire the lock and start its watchdog.
func (l *RedisLocker) acquireOnce(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	success, err := l.client.SetNX(ctx, l.lockName, l.ownerID, l.lockTimeout).Result()
	if err != nil {
		l.logger.Error("Failed to set lock", "error", err)
		return err
	}
	if success {
		l.startWatchdog(ctx)
		l.logger.Info("Lock acquired", "ownerID", l.ownerID)
		return nil
	}

	currentOwnerID, err := l.client.Get(ctx, l.lockName).Result()
	if err != nil {
		l.logger.Error("Failed to get current owner ID", "error", err)
		return err
	}
	if currentOwnerID == l.ownerID {
		// Already held by the current owner (reentrant); treat as success.
		l.logger.Info("Lock is already held by the current owner", "ownerID", l.ownerID)
		return nil
	}

	return fmt.Errorf("%w: %s", ErrLockHeld, currentOwnerID)
}

// waitRetry blocks for the configured delay before the next acquisition attempt,
// returning early if ctx is cancelled.
func (l *RedisLocker) waitRetry(ctx context.Context, attempt int) error {
	timer := time.NewTimer(l.delayFunc(attempt))
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Unlock releases the distributed lock, but only if it is still held by this owner.
func (l *RedisLocker) Unlock(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.stopWatchdog()

	released, err := unlockScript.Run(ctx, l.client, []string{l.lockName}, l.ownerID).Int()
	if err != nil {
		l.logger.Error("Failed to delete lock", "error", err)
		return err
	}
	if released == 0 {
		// The lock was already released or is now held by another owner.
		l.logger.Warn("Lock already released or not held by current owner", "lockName", l.lockName)
		return nil
	}

	l.logger.Info("Lock released", "ownerID", l.ownerID)
	return nil
}

// Renew refreshes the lock's expiration time, but only if it is still held by this owner.
func (l *RedisLocker) Renew(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	renewed, err := renewScript.Run(ctx, l.client, []string{l.lockName}, l.ownerID, l.lockTimeout.Milliseconds()).Int()
	if err != nil {
		l.logger.Error("Failed to renew lock", "error", err)
		return err
	}
	if renewed == 0 {
		l.logger.Warn("Renew failed: lock not held by current owner", "lockName", l.lockName)
		return ErrLockNotHeld
	}

	l.logger.Info("Lock renewed", "ownerID", l.ownerID)
	return nil
}

// startWatchdog launches a goroutine that periodically renews the lock until
// cancelled. It must be called while holding l.mu.
func (l *RedisLocker) startWatchdog(ctx context.Context) {
	watchCtx, cancel := context.WithCancel(ctx)
	l.cancel = cancel
	go l.renewLoop(watchCtx)
}

// stopWatchdog stops the renewal goroutine. It must be called while holding l.mu.
func (l *RedisLocker) stopWatchdog() {
	if l.cancel != nil {
		l.cancel()
		l.cancel = nil
	}
}

// renewLoop periodically renews the lock and stops when the lock is lost or the
// onExtend callback requests a stop.
func (l *RedisLocker) renewLoop(ctx context.Context) {
	ticker := time.NewTicker(l.lockTimeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := l.Renew(ctx); err != nil {
				if errors.Is(err, ErrLockNotHeld) {
					l.logger.Warn("Lost lock ownership, stopping renewal", "lockName", l.lockName)
					return
				}
				l.logger.Error("Failed to renew lock", "error", err)
				continue
			}
			if err := l.onExtend(); err != nil {
				l.logger.Error("OnExtend callback failed, stopping renewal", "error", err)
				return
			}
		}
	}
}
