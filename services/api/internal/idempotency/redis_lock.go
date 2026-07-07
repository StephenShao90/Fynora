package idempotency

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrInFlight = errors.New("idempotency key is already in flight")

type LockStore interface {
	Acquire(ctx context.Context, key, requestHash string, ttl time.Duration) (bool, error)
	Release(ctx context.Context, key string) error
}

type MemoryLockStore struct {
	mu    sync.Mutex
	locks map[string]memoryLock
	now   func() time.Time
}

type memoryLock struct {
	requestHash string
	expiresAt   time.Time
}

func NewMemoryLockStore() *MemoryLockStore {
	return &MemoryLockStore{locks: map[string]memoryLock{}, now: time.Now}
}

func (s *MemoryLockStore) Acquire(ctx context.Context, key, requestHash string, ttl time.Duration) (bool, error) {
	now := s.now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	if lock, ok := s.locks[key]; ok && now.Before(lock.expiresAt) {
		return false, nil
	}
	s.locks[key] = memoryLock{requestHash: requestHash, expiresAt: now.Add(ttl)}
	return true, nil
}

func (s *MemoryLockStore) Release(ctx context.Context, key string) error {
	s.mu.Lock()
	delete(s.locks, key)
	s.mu.Unlock()
	return nil
}

type RedisLockClient interface {
	SetNX(ctx context.Context, key, value string, ttl time.Duration) (bool, error)
	Del(ctx context.Context, key string) error
}

type RedisLockStore struct {
	client RedisLockClient
}

func NewRedisLockStore(client RedisLockClient) *RedisLockStore {
	return &RedisLockStore{client: client}
}

func (s *RedisLockStore) Acquire(ctx context.Context, key, requestHash string, ttl time.Duration) (bool, error) {
	return s.client.SetNX(ctx, key, requestHash, ttl)
}

func (s *RedisLockStore) Release(ctx context.Context, key string) error {
	return s.client.Del(ctx, key)
}
