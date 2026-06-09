package dedupe

import (
	"context"
	"fmt"
	"time"

	"github.com/austin72905/go-infra/app"
	"github.com/austin72905/go-infra/redis"
)

const DefaultComponentName = "Deduplicator"

type Component struct {
	appRuntime         *app.Runtime
	redisComponentName string
	manager            *Manager
}

func (d *Component) Initialize(runtime *app.Runtime, name string) {
	d.appRuntime = runtime
}

func (d *Component) Validate() {
}

func (d *Component) TryMark(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return d.ensureManager().TryMark(ctx, key, ttl)
}

func (d *Component) Marked(ctx context.Context, key string) (bool, error) {
	return d.ensureManager().Marked(ctx, key)
}

func (d *Component) Clear(ctx context.Context, key string) error {
	return d.ensureManager().Clear(ctx, key)
}

func (d *Component) SetRedisComponentName(name string) {
	d.redisComponentName = name
}

func (d *Component) ensureManager() *Manager {
	if d.manager != nil {
		return d.manager
	}

	component, ok := d.appRuntime.Components.LookupComponent(d.redisComponentName)
	if !ok {
		panic(fmt.Sprintf("redis component %q is not registered", d.redisComponentName))
	}

	redisComponent, typeOK := component.(*redis.Component)
	if !typeOK {
		panic(fmt.Sprintf("component %q is not a redis.Component", d.redisComponentName))
	}

	d.manager = NewManager(NewRedisBackend(redisComponent.Client().Raw()))
	return d.manager
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
