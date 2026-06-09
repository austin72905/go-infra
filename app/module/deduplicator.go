package module

import (
	"context"
	"fmt"
	"time"

	internaldedupe "github.com/austin72905/go-infra/internal/dedupe"
)

const defaultDeduplicatorName = "Deduplicator"

type (
	DedupeError = internaldedupe.DedupeError
)

var (
	ErrDedupeKeyRequired = internaldedupe.ErrDedupeKeyRequired
	ErrDedupeTTLRequired = internaldedupe.ErrDedupeTTLRequired
)

type Deduplicator struct {
	appRuntime         *AppRuntime
	redisComponentName string
	manager            *internaldedupe.Manager
}

func (d *Deduplicator) initialize(runtime *AppRuntime, name string) {
	d.appRuntime = runtime
}

func (d *Deduplicator) validate() {
}

func (d *Deduplicator) TryMark(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return d.ensureManager().TryMark(ctx, key, ttl)
}

func (d *Deduplicator) Marked(ctx context.Context, key string) (bool, error) {
	return d.ensureManager().Marked(ctx, key)
}

func (d *Deduplicator) Clear(ctx context.Context, key string) error {
	return d.ensureManager().Clear(ctx, key)
}

func (d *Deduplicator) ensureManager() *internaldedupe.Manager {
	if d.manager != nil {
		return d.manager
	}

	component, ok := d.appRuntime.Components.LookupComponent(d.redisComponentName)
	if !ok {
		panic(fmt.Sprintf("redis component %q is not registered", d.redisComponentName))
	}

	redisComponent, typeOK := component.(*RedisComponent)
	if !typeOK {
		panic(fmt.Sprintf("component %q is not a RedisComponent", d.redisComponentName))
	}

	d.manager = internaldedupe.NewManager(internaldedupe.NewRedisBackend(redisComponent.Client().Raw()))
	return d.manager
}

func deduplicatorName(name string) string {
	if name == "" {
		return defaultDeduplicatorName
	}
	return defaultDeduplicatorName + ":" + name
}

func deduplicatorRedisComponentName(name string) string {
	if name == "" {
		return defaultRedisComponentName
	}
	return redisNamedComponentName(name)
}
