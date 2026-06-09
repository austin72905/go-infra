package redislock

import (
	"context"
	"fmt"
	"time"

	"github.com/austin72905/go-infra/app"
	"github.com/austin72905/go-infra/redis"
)

const DefaultComponentName = "RedisLock"

type Component struct {
	appRuntime         *app.Runtime
	redisComponentName string
	manager            *Manager
}

func (rm *Component) Initialize(runtime *app.Runtime, name string) {
	rm.appRuntime = runtime
}

func (rm *Component) Validate() {
}

func (rm *Component) TryLock(ctx context.Context, key string, ttl time.Duration) (*Lock, bool, error) {
	return rm.ensureManager().TryLock(ctx, key, ttl)
}

func (rm *Component) Acquire(ctx context.Context, key string, ttl time.Duration, opts ...Option) (*Lock, error) {
	return rm.ensureManager().Acquire(ctx, key, ttl, opts...)
}

func (rm *Component) WithLock(ctx context.Context, key string, ttl time.Duration, fn func(ctx context.Context) error, opts ...Option) error {
	return rm.ensureManager().WithLock(ctx, key, ttl, fn, opts...)
}

func (rm *Component) SetRedisComponentName(name string) {
	rm.redisComponentName = name
}

func (rm *Component) ensureManager() *Manager {
	if rm.manager != nil {
		return rm.manager
	}

	component, ok := rm.appRuntime.Components.LookupComponent(rm.redisComponentName)
	if !ok {
		panic(fmt.Sprintf("redis component %q is not registered", rm.redisComponentName))
	}

	redisComponent, typeOK := component.(*redis.Component)
	if !typeOK {
		panic(fmt.Sprintf("component %q is not a redis.Component", rm.redisComponentName))
	}

	rm.manager = NewManager(NewRedisBackend(redisComponent.Client().Raw()))
	return rm.manager
}

func ComponentName(name string) string {
	if name == "" {
		return DefaultComponentName
	}
	return DefaultComponentName + ":" + name
}

func RedisComponentName(name string) string {
	if name == "" {
		return redis.DefaultComponentName
	}
	return redis.ComponentName(name)
}
