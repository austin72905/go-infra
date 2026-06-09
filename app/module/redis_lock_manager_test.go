package module

import "testing"

func TestModuleBaseRedisNamedInstance(t *testing.T) {
	t.Parallel()

	runtime := NewAppRuntime()
	moduleBase := &ModuleBase{AppRuntime: runtime}

	defaultRedis := moduleBase.Redis()
	if defaultRedis == nil {
		t.Fatal("expected default redis component")
	}

	namedRedis := moduleBase.Redis("cache")
	if namedRedis == nil {
		t.Fatal("expected named redis component")
	}
	if namedRedis == defaultRedis {
		t.Fatal("expected named redis component to be different from default redis component")
	}

	again := moduleBase.Redis("cache")
	if again != namedRedis {
		t.Fatal("expected repeated named redis lookup to return same component")
	}
}

func TestModuleBaseRedisLockUsesNamedRedisComponent(t *testing.T) {
	t.Parallel()

	runtime := NewAppRuntime()
	moduleBase := &ModuleBase{AppRuntime: runtime}

	lockManager := moduleBase.RedisLock("cache")
	if lockManager == nil {
		t.Fatal("expected redis lock manager")
	}
	if lockManager.redisComponentName != redisNamedComponentName("cache") {
		t.Fatalf("expected lock manager to use named redis component, got %q", lockManager.redisComponentName)
	}

	component, ok := runtime.Components.LookupComponent(redisNamedComponentName("cache"))
	if !ok {
		t.Fatal("expected named redis component to be registered")
	}
	if _, typeOK := component.(*RedisComponent); !typeOK {
		t.Fatal("expected registered component to be a RedisComponent")
	}

	again := moduleBase.RedisLock("cache")
	if again != lockManager {
		t.Fatal("expected repeated named redis lock lookup to return same component")
	}
}

func TestModuleBaseDeduplicatorUsesNamedRedisComponent(t *testing.T) {
	t.Parallel()

	runtime := NewAppRuntime()
	moduleBase := &ModuleBase{AppRuntime: runtime}

	deduplicator := moduleBase.Deduplicator("cache")
	if deduplicator == nil {
		t.Fatal("expected deduplicator")
	}
	if deduplicator.redisComponentName != redisNamedComponentName("cache") {
		t.Fatalf("expected deduplicator to use named redis component, got %q", deduplicator.redisComponentName)
	}

	component, ok := runtime.Components.LookupComponent(redisNamedComponentName("cache"))
	if !ok {
		t.Fatal("expected named redis component to be registered")
	}
	if _, typeOK := component.(*RedisComponent); !typeOK {
		t.Fatal("expected registered component to be a RedisComponent")
	}

	again := moduleBase.Deduplicator("cache")
	if again != deduplicator {
		t.Fatal("expected repeated named deduplicator lookup to return same component")
	}
}
