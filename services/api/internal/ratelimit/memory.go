package ratelimit

import (
	"context"
	"sync"
	"time"
)

type MemoryLimiter struct {
	mu      sync.Mutex
	buckets map[string]bucket
	now     func() time.Time
}

type bucket struct {
	count   int
	resetAt time.Time
}

func NewMemoryLimiter() *MemoryLimiter {
	return &MemoryLimiter{buckets: map[string]bucket{}, now: time.Now}
}

func (l *MemoryLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (Decision, error) {
	now := l.now().UTC()
	l.mu.Lock()
	defer l.mu.Unlock()
	b := l.buckets[key]
	if b.resetAt.IsZero() || now.After(b.resetAt) {
		b = bucket{resetAt: now.Add(window)}
	}
	b.count++
	l.buckets[key] = b
	retry := time.Until(b.resetAt)
	if retry < time.Second {
		retry = time.Second
	}
	return Decision{Allowed: b.count <= limit, RetryAfter: retry}, nil
}
