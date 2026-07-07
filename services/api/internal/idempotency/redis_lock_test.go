package idempotency

import (
	"context"
	"testing"
	"time"
)

func TestMemoryLockStorePreventsDuplicateInFlight(t *testing.T) {
	store := NewMemoryLockStore()
	ok, err := store.Acquire(context.Background(), "key", "hash", time.Minute)
	if err != nil || !ok {
		t.Fatalf("expected first acquire: ok=%v err=%v", ok, err)
	}
	ok, err = store.Acquire(context.Background(), "key", "hash", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected duplicate lock to be rejected")
	}
	if err := store.Release(context.Background(), "key"); err != nil {
		t.Fatal(err)
	}
	ok, err = store.Acquire(context.Background(), "key", "hash", time.Minute)
	if err != nil || !ok {
		t.Fatalf("expected acquire after release: ok=%v err=%v", ok, err)
	}
}
