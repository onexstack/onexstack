package distlock

import (
	"testing"
	"time"
)

func TestNewOwnerIDUnique(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		id := newOwnerID()
		if id == "" {
			t.Fatal("newOwnerID() returned an empty string")
		}
		if _, ok := seen[id]; ok {
			t.Fatalf("newOwnerID() returned a duplicate value %q", id)
		}
		seen[id] = struct{}{}
	}
}

func TestDefaultDelayFuncBounds(t *testing.T) {
	for attempt := 1; attempt <= 20; attempt++ {
		d := defaultDelayFunc(attempt)
		if d < 0 {
			t.Fatalf("defaultDelayFunc(%d) = %v, want >= 0", attempt, d)
		}
		if d >= 512*time.Millisecond {
			t.Fatalf("defaultDelayFunc(%d) = %v, want < 512ms", attempt, d)
		}
	}
}

func TestNewOptionsDefaults(t *testing.T) {
	o := NewOptions()

	if o.lockName != DefaultLockName {
		t.Fatalf("lockName = %q, want %q", o.lockName, DefaultLockName)
	}
	if o.lockTimeout != 10*time.Second {
		t.Fatalf("lockTimeout = %v, want 10s", o.lockTimeout)
	}
	if o.tries != 1 {
		t.Fatalf("tries = %d, want 1", o.tries)
	}
	if o.ownerID == "" {
		t.Fatal("ownerID should not be empty")
	}
	if o.delayFunc == nil {
		t.Fatal("delayFunc should not be nil")
	}
	if o.onExtend == nil {
		t.Fatal("onExtend should not be nil")
	}
}
