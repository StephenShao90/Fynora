package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestMemoryLimiterAllowsUntilLimit(t *testing.T) {
	limiter := NewMemoryLimiter()
	for i := 0; i < 2; i++ {
		decision, err := limiter.Allow(context.Background(), "auth:1", 2, time.Minute)
		if err != nil || !decision.Allowed {
			t.Fatalf("expected allowed decision %d: %#v err=%v", i, decision, err)
		}
	}
	decision, err := limiter.Allow(context.Background(), "auth:1", 2, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Allowed {
		t.Fatal("expected request to be rate limited")
	}
}

type fakeCounter struct{ count int64 }

func (f *fakeCounter) IncrExpire(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	f.count++
	return f.count, nil
}

func TestRedisLimiterUsesCounter(t *testing.T) {
	counter := &fakeCounter{}
	limiter := NewRedisLimiter(counter)
	first, err := limiter.Allow(context.Background(), "heavy:1", 1, time.Minute)
	if err != nil || !first.Allowed {
		t.Fatalf("expected first allowed: %#v err=%v", first, err)
	}
	second, err := limiter.Allow(context.Background(), "heavy:1", 1, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if second.Allowed {
		t.Fatal("expected second call limited")
	}
}
