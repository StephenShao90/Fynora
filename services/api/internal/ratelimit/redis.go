package ratelimit

import (
	"context"
	"time"
)

type RedisCounter interface {
	IncrExpire(ctx context.Context, key string, ttl time.Duration) (int64, error)
}

type RedisLimiter struct {
	redis RedisCounter
}

func NewRedisLimiter(redis RedisCounter) *RedisLimiter {
	return &RedisLimiter{redis: redis}
}

func (l *RedisLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (Decision, error) {
	count, err := l.redis.IncrExpire(ctx, key, window)
	if err != nil {
		return Decision{}, err
	}
	return Decision{Allowed: count <= int64(limit), RetryAfter: window}, nil
}
