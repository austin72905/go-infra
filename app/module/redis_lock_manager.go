package module

import (
	"context"
	"fmt"
	"time"

	internalredislock "go-infra/internal/redislock"
)

const defaultRedisLockManagerName = "RedisLock"

type (
	Lock          = internalredislock.Lock
	LockOption    = internalredislock.Option
	RetryStrategy = internalredislock.RetryStrategy
	LockError     = internalredislock.LockError
)

var (
	ErrLockNotAcquired = internalredislock.ErrLockNotAcquired
	ErrLockNotHeld     = internalredislock.ErrLockNotHeld
	ErrLockReleased    = internalredislock.ErrLockReleased
)

func WithRetryStrategy(strategy RetryStrategy) LockOption {
	return internalredislock.WithRetryStrategy(strategy)
}

func WithMaxRetries(maxRetries int) LockOption {
	return internalredislock.WithMaxRetries(maxRetries)
}

func WithBackoff(backoff time.Duration) LockOption {
	return internalredislock.WithBackoff(backoff)
}

func WithAutoRefresh() LockOption {
	return internalredislock.WithAutoRefresh()
}

func FixedBackoff(backoff time.Duration, maxRetries int) RetryStrategy {
	return internalredislock.FixedBackoff(backoff, maxRetries)
}

func LinearBackoff(step time.Duration, maxRetries int) RetryStrategy {
	return internalredislock.LinearBackoff(step, maxRetries)
}

type RedisLockManager struct {
	appRuntime         *AppRuntime
	redisComponentName string
	manager            *internalredislock.Manager
}

func (rm *RedisLockManager) initialize(runtime *AppRuntime, name string) {
	rm.appRuntime = runtime
}

func (rm *RedisLockManager) validate() {
}

func (rm *RedisLockManager) TryLock(ctx context.Context, key string, ttl time.Duration) (*Lock, bool, error) {
	return rm.ensureManager().TryLock(ctx, key, ttl)
}

func (rm *RedisLockManager) Acquire(ctx context.Context, key string, ttl time.Duration, opts ...LockOption) (*Lock, error) {
	return rm.ensureManager().Acquire(ctx, key, ttl, opts...)
}

func (rm *RedisLockManager) WithLock(ctx context.Context, key string, ttl time.Duration, fn func(ctx context.Context) error, opts ...LockOption) error {
	return rm.ensureManager().WithLock(ctx, key, ttl, fn, opts...)
}

func (rm *RedisLockManager) ensureManager() *internalredislock.Manager {
	if rm.manager != nil {
		return rm.manager
	}

	component, ok := rm.appRuntime.Components.LookupComponent(rm.redisComponentName)
	if !ok {
		panic(fmt.Sprintf("redis component %q is not registered", rm.redisComponentName))
	}

	redisComponent, typeOK := component.(*RedisComponent)
	if !typeOK {
		panic(fmt.Sprintf("component %q is not a RedisComponent", rm.redisComponentName))
	}

	rm.manager = internalredislock.NewManager(internalredislock.NewRedisBackend(redisComponent.Client().Raw()))
	return rm.manager
}

func redisLockManagerName(name string) string {
	if name == "" {
		return defaultRedisLockManagerName
	}
	return defaultRedisLockManagerName + ":" + name
}

func redisLockRedisComponentName(name string) string {
	if name == "" {
		return defaultRedisComponentName
	}
	return redisNamedComponentName(name)
}
