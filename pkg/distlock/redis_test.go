//go:build redis

// This file requires github.com/alicebob/miniredis/v2 and an environment able to
// fetch it from the module proxy. Run it with:
//
//	go get github.com/alicebob/miniredis/v2
//	go test -tags redis -race ./pkg/distlock/...
package distlock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// newTestRedis starts an in-memory miniredis server and returns it together
// with a connected go-redis client.
func newTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()

	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	return s, client
}

func TestRedisLockerLockUnlock(t *testing.T) {
	_, client := newTestRedis(t)
	ctx := context.Background()
	locker := NewRedisLocker(client, WithLockName("lk"), WithOwnerID("A"))

	if err := locker.Lock(ctx); err != nil {
		t.Fatalf("Lock() = %v, want nil", err)
	}

	// The lock key must exist with our token as its value.
	if val, err := client.Get(ctx, "lk").Result(); err != nil || val != "A" {
		t.Fatalf("Get() = (%q, %v), want (A, nil)", val, err)
	}

	if err := locker.Unlock(ctx); err != nil {
		t.Fatalf("Unlock() = %v, want nil", err)
	}

	// After unlock the key must be gone.
	if err := client.Get(ctx, "lk").Err(); !errors.Is(err, redis.Nil) {
		t.Fatalf("Get() after Unlock err = %v, want redis.Nil", err)
	}

	// Unlock is idempotent: a second call must not error.
	if err := locker.Unlock(ctx); err != nil {
		t.Fatalf("second Unlock() = %v, want nil", err)
	}
}

func TestRedisLockerReentrantAndContention(t *testing.T) {
	_, client := newTestRedis(t)
	ctx := context.Background()
	a := NewRedisLocker(client, WithLockName("lk"), WithOwnerID("A"))
	b := NewRedisLocker(client, WithLockName("lk"), WithOwnerID("B"))

	if err := a.Lock(ctx); err != nil {
		t.Fatalf("a.Lock() = %v, want nil", err)
	}
	defer a.Unlock(ctx)

	// Reentrant acquire by the same owner must succeed.
	if err := a.Lock(ctx); err != nil {
		t.Fatalf("reentrant a.Lock() = %v, want nil", err)
	}

	// A different owner must fail with ErrLockHeld.
	if err := b.Lock(ctx); !errors.Is(err, ErrLockHeld) {
		t.Fatalf("b.Lock() = %v, want ErrLockHeld", err)
	}
}

func TestRedisLockerUnlockDoesNotDeleteOthersLock(t *testing.T) {
	_, client := newTestRedis(t)
	ctx := context.Background()
	a := NewRedisLocker(client, WithLockName("lk"), WithOwnerID("A"))

	if err := a.Lock(ctx); err != nil {
		t.Fatalf("Lock() = %v, want nil", err)
	}

	// Simulate A losing the lock to B (e.g. A's lock expired and B acquired it).
	if err := client.Set(ctx, "lk", "B", time.Minute).Err(); err != nil {
		t.Fatalf("Set() = %v, want nil", err)
	}

	// A's unlock must not delete B's lock.
	if err := a.Unlock(ctx); err != nil {
		t.Fatalf("Unlock() = %v, want nil", err)
	}

	if val, err := client.Get(ctx, "lk").Result(); err != nil || val != "B" {
		t.Fatalf("Get() = (%q, %v), want (B, nil): A deleted B's lock", val, err)
	}
}

func TestRedisLockerRenewNotHeld(t *testing.T) {
	_, client := newTestRedis(t)
	ctx := context.Background()
	a := NewRedisLocker(client, WithLockName("lk"), WithOwnerID("A"))

	if err := a.Lock(ctx); err != nil {
		t.Fatalf("Lock() = %v, want nil", err)
	}

	// A loses the lock to B.
	if err := client.Set(ctx, "lk", "B", time.Minute).Err(); err != nil {
		t.Fatalf("Set() = %v, want nil", err)
	}

	if err := a.Renew(ctx); !errors.Is(err, ErrLockNotHeld) {
		t.Fatalf("Renew() = %v, want ErrLockNotHeld", err)
	}

	// B's lock must remain intact.
	if val, err := client.Get(ctx, "lk").Result(); err != nil || val != "B" {
		t.Fatalf("Get() = (%q, %v), want (B, nil)", val, err)
	}
}

func TestRedisLockerRenewExtendsTTL(t *testing.T) {
	s, client := newTestRedis(t)
	ctx := context.Background()
	a := NewRedisLocker(client, WithLockName("lk"), WithOwnerID("A"), WithLockTimeout(10*time.Second))

	if err := a.Lock(ctx); err != nil {
		t.Fatalf("Lock() = %v, want nil", err)
	}

	// Advance the server clock so the TTL is partially consumed.
	s.FastForward(5 * time.Second)

	if err := a.Renew(ctx); err != nil {
		t.Fatalf("Renew() = %v, want nil", err)
	}

	ttl, err := client.TTL(ctx, "lk").Result()
	if err != nil {
		t.Fatalf("TTL() = %v, want nil", err)
	}
	if ttl < 9*time.Second {
		t.Fatalf("TTL() = %v, want ~10s after Renew", ttl)
	}
}

func TestRedisLockerRetryGivesUp(t *testing.T) {
	_, client := newTestRedis(t)
	ctx := context.Background()

	holder := NewRedisLocker(client, WithLockName("lk"), WithOwnerID("holder"))
	if err := holder.Lock(ctx); err != nil {
		t.Fatalf("holder.Lock() = %v, want nil", err)
	}
	defer holder.Unlock(ctx)

	a := NewRedisLocker(client,
		WithLockName("lk"),
		WithOwnerID("A"),
		WithTries(2),
		WithRetryDelay(time.Millisecond),
	)

	if err := a.Lock(ctx); !errors.Is(err, ErrLockHeld) {
		t.Fatalf("a.Lock() = %v, want ErrLockHeld", err)
	}
}

func TestRedisLockerRetrySucceedsAfterRelease(t *testing.T) {
	_, client := newTestRedis(t)
	ctx := context.Background()

	holder := NewRedisLocker(client, WithLockName("lk"), WithOwnerID("holder"))
	if err := holder.Lock(ctx); err != nil {
		t.Fatalf("holder.Lock() = %v, want nil", err)
	}

	a := NewRedisLocker(client,
		WithLockName("lk"),
		WithOwnerID("A"),
		WithTries(10),
		WithRetryDelay(10*time.Millisecond),
	)

	// Release the holder after a short delay so A's retry eventually succeeds.
	go func() {
		time.Sleep(40 * time.Millisecond)
		_ = holder.Unlock(context.Background())
	}()

	if err := a.Lock(ctx); err != nil {
		t.Fatalf("a.Lock() = %v, want nil after holder released", err)
	}
	defer a.Unlock(ctx)
}

func TestRedisLockerOnExtend(t *testing.T) {
	_, client := newTestRedis(t)
	ctx := context.Background()

	called := make(chan struct{}, 2)
	a := NewRedisLocker(client,
		WithLockName("lk"),
		WithOwnerID("A"),
		WithLockTimeout(100*time.Millisecond),
		WithOnExtendFunc(func() error {
			called <- struct{}{}
			return errors.New("stop")
		}),
	)

	if err := a.Lock(ctx); err != nil {
		t.Fatalf("Lock() = %v, want nil", err)
	}
	defer a.Unlock(ctx)

	select {
	case <-called:
		// The callback was invoked by the watchdog.
	case <-time.After(2 * time.Second):
		t.Fatal("onExtend callback was not called")
	}
}
