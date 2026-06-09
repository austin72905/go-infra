package redis

import (
	"context"
	"fmt"
	"strconv"

	"github.com/austin72905/go-infra/app"
	internalmodule "github.com/austin72905/go-infra/internal/module"
)

const DefaultComponentName = "Redis"

type Component struct {
	appRuntime         *app.Runtime
	client             *Client
	addr               string
	db                 int
	password           string
	shutdownRegistered bool
}

func (rc *Component) Initialize(runtime *app.Runtime, name string) {
	rc.appRuntime = runtime
	if rc.shutdownRegistered {
		return
	}

	runtime.Lifecycle.Shutdown.AddReleaseResources(internalmodule.TaskFunc(func(ctx context.Context) {
		_ = rc.Close()
	}))
	rc.shutdownRegistered = true
}

func (rc *Component) Validate() {
}

func (rc *Component) SetAddr(addr string) {
	rc.addr = addr
	rc.appRuntime.Probe.AddHostURI(addr)
}

func (rc *Component) SetDB(db int) {
	rc.db = db
}

func (rc *Component) SetPassword(password string) {
	rc.password = password
}

func (rc *Component) LoadFromProperties(addrKey, dbKey, passwordKey string) {
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

func (rc *Component) LoadFromPrefix(prefix string) {
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

func (rc *Component) EnsureClient() {
	if rc.client != nil {
		return
	}
	if rc.addr == "" {
		panic("redis addr is required")
	}
	rc.client = New(rc.addr, rc.db, rc.password)
}

func (rc *Component) Client() *Client {
	rc.EnsureClient()
	return rc.client
}

func (rc *Component) Ping(ctx context.Context) error {
	return rc.Client().Ping(ctx)
}

func (rc *Component) Close() error {
	if rc.client == nil {
		return nil
	}
	err := rc.client.Close()
	rc.client = nil
	return err
}

func (rc *Component) MustPing(ctx context.Context) {
	if err := rc.Ping(ctx); err != nil {
		panic(fmt.Sprintf("redis ping failed: %v", err))
	}
}

func ComponentName(name string) string {
	if name == "" {
		return DefaultComponentName
	}
	return DefaultComponentName + ":" + name
}
