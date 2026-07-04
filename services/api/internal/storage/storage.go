package storage

import (
	"context"
	"os"
	"path/filepath"
)

type RawEventStore interface {
	Put(ctx context.Context, key string, payload []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
}

type LocalStore struct{ Dir string }

func NewLocalStore(dir string) LocalStore {
	return LocalStore{Dir: dir}
}

func (s LocalStore) Put(ctx context.Context, key string, payload []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	path := filepath.Join(s.Dir, filepath.Clean(key))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o644)
}

func (s LocalStore) Get(ctx context.Context, key string) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return os.ReadFile(filepath.Join(s.Dir, filepath.Clean(key)))
}

type S3Store struct{}

func (S3Store) Put(context.Context, string, []byte) error {
	return ErrS3NotConfigured
}

func (S3Store) Get(context.Context, string) ([]byte, error) {
	return nil, ErrS3NotConfigured
}

var ErrS3NotConfigured = os.ErrInvalid
