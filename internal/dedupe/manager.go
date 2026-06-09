package dedupe

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var (
	ErrDedupeKeyRequired = errors.New("dedupe key is required")
	ErrDedupeTTLRequired = errors.New("dedupe ttl must be greater than zero")
)

type DedupeError struct {
	Op    string
	Key   string
	Cause error
	Msg   string
}

func (e *DedupeError) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	if e.Cause != nil {
		return fmt.Sprintf("dedupe %s failed for key %q: %v", e.Op, e.Key, e.Cause)
	}
	return fmt.Sprintf("dedupe %s failed for key %q", e.Op, e.Key)
}

func (e *DedupeError) Unwrap() error {
	return e.Cause
}

type Backend interface {
	TryMark(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Marked(ctx context.Context, key string) (bool, error)
	Clear(ctx context.Context, key string) error
}

type RedisBackend struct {
	client *goredis.Client
}

func NewRedisBackend(client *goredis.Client) *RedisBackend {
	return &RedisBackend{client: client}
}

func (b *RedisBackend) TryMark(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return b.client.SetNX(ctx, key, "1", ttl).Result()
}

func (b *RedisBackend) Marked(ctx context.Context, key string) (bool, error) {
	result, err := b.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

func (b *RedisBackend) Clear(ctx context.Context, key string) error {
	return b.client.Del(ctx, key).Err()
}

type Manager struct {
	backend Backend
}

func NewManager(backend Backend) *Manager {
	return &Manager{backend: backend}
}

func (m *Manager) TryMark(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if key == "" {
		return false, &DedupeError{Op: "tryMark", Key: key, Cause: ErrDedupeKeyRequired}
	}
	if ttl <= 0 {
		return false, &DedupeError{Op: "tryMark", Key: key, Cause: ErrDedupeTTLRequired}
	}

	ok, err := m.backend.TryMark(ctx, key, ttl)
	if err != nil {
		return false, &DedupeError{Op: "tryMark", Key: key, Cause: err}
	}
	return ok, nil
}

func (m *Manager) Marked(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, &DedupeError{Op: "marked", Key: key, Cause: ErrDedupeKeyRequired}
	}

	ok, err := m.backend.Marked(ctx, key)
	if err != nil {
		return false, &DedupeError{Op: "marked", Key: key, Cause: err}
	}
	return ok, nil
}

func (m *Manager) Clear(ctx context.Context, key string) error {
	if key == "" {
		return &DedupeError{Op: "clear", Key: key, Cause: ErrDedupeKeyRequired}
	}

	if err := m.backend.Clear(ctx, key); err != nil {
		return &DedupeError{Op: "clear", Key: key, Cause: err}
	}
	return nil
}
