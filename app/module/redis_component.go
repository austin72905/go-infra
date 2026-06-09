package module

import (
	"context"
	"fmt"
	"strconv"

	internalmodule "go-infra/internal/module"
	internalredis "go-infra/internal/redis"
)

const redisComponentName = "Redis"
const defaultRedisComponentName = redisComponentName

// Redis 只做了「連線與框架整合」
type RedisComponent struct {
	appRuntime         *AppRuntime
	client             *internalredis.Client
	addr               string
	db                 int
	password           string
	shutdownRegistered bool
}

func (rc *RedisComponent) initialize(runtime *AppRuntime, name string) {
	rc.appRuntime = runtime
	if rc.shutdownRegistered {
		return
	}

	runtime.Lifecycle.Shutdown.AddReleaseResources(internalmodule.TaskFunc(func(ctx context.Context) {
		_ = rc.Close()
	}))
	rc.shutdownRegistered = true
}

func (rc *RedisComponent) validate() {
}

func (rc *RedisComponent) SetAddr(addr string) {
	rc.addr = addr
	rc.appRuntime.Probe.AddHostURI(addr)
}

func (rc *RedisComponent) SetDB(db int) {
	rc.db = db
}

func (rc *RedisComponent) SetPassword(password string) {
	rc.password = password
}

func (rc *RedisComponent) LoadFromProperties(addrKey, dbKey, passwordKey string) {
	rc.SetAddr(rc.appRuntime.Property.RequiredProperty(addrKey))

	if dbKey != "" {
		dbValue := rc.appRuntime.Property.Property(dbKey)
		if dbValue != "" {
			var parsed int
			_, err := fmt.Sscanf(dbValue, "%d", &parsed)
			if err != nil {
				panic(fmt.Sprintf("invalid redis db value for key %q: %v", dbKey, err))
			}
			rc.SetDB(parsed)
		}
	}

	if passwordKey != "" {
		rc.SetPassword(rc.appRuntime.Property.Property(passwordKey))
	}
}

/*
prefix + ".addr"
prefix + ".db"
prefix + ".password"

redis.addr=localhost:6379
redis.db=0
redis.password=
*/
func (rc *RedisComponent) LoadFromPrefix(prefix string) {
	if prefix == "" {
		panic("redis property prefix is required")
	}

	rc.SetAddr(rc.appRuntime.Property.RequiredProperty(prefix + ".addr"))

	dbKey := prefix + ".db"
	dbValue := rc.appRuntime.Property.Property(dbKey)
	if dbValue != "" {
		parsed, err := strconv.Atoi(dbValue)
		if err != nil {
			panic(fmt.Sprintf("invalid redis db value for key %q: %v", dbKey, err))
		}
		rc.SetDB(parsed)
	}

	rc.SetPassword(rc.appRuntime.Property.Property(prefix + ".password"))
}

func (rc *RedisComponent) ensureClient() {
	if rc.client != nil {
		return
	}
	if rc.addr == "" {
		panic("redis addr is required")
	}
	rc.client = internalredis.New(rc.addr, rc.db, rc.password)
}

func (rc *RedisComponent) Client() *internalredis.Client {
	rc.ensureClient()
	return rc.client
}

func (rc *RedisComponent) Ping(ctx context.Context) error {
	return rc.Client().Ping(ctx)
}

func (rc *RedisComponent) Close() error {
	if rc.client == nil {
		return nil
	}
	err := rc.client.Close()
	rc.client = nil
	return err
}

func (rc *RedisComponent) MustPing(ctx context.Context) {
	if err := rc.Ping(ctx); err != nil {
		panic(fmt.Sprintf("redis ping failed: %v", err))
	}
}

func redisNamedComponentName(name string) string {
	if name == "" {
		return defaultRedisComponentName
	}
	return defaultRedisComponentName + ":" + name
}
