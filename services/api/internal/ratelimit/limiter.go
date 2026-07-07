package ratelimit

import (
	"context"
	"time"
)

type Decision struct {
	Allowed    bool
	RetryAfter time.Duration
}

type Limiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (Decision, error)
}
