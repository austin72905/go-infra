package module

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"strconv"
	"time"

	internalmodule "go-infra/internal/module"
	internalmongo "go-infra/internal/mongo"

	orgmongo "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const defaultMongoComponentName = "Mongo"

type MongoComponent struct {
	appRuntime *AppRuntime
	client     *internalmongo.Client

	uri                    string
	readPreference         *readpref.ReadPref
	timeout                time.Duration
	minPoolSize            uint64
	maxPoolSize            uint64
	maxConnecting          uint64
	tlsConfig              *tls.Config
	slowOperationThreshold time.Duration
	shutdownRegistered     bool
}

func (mc *MongoComponent) initialize(runtime *AppRuntime, name string) {
	mc.appRuntime = runtime
	if mc.shutdownRegistered {
		return
	}

	runtime.Lifecycle.Shutdown.AddReleaseResources(internalmodule.TaskFunc(func(ctx context.Context) {
		_ = mc.Close()
	}))
	mc.shutdownRegistered = true
}

func (mc *MongoComponent) validate() {
}

func (mc *MongoComponent) SetURI(uri string) *MongoComponent {
	mc.uri = uri
	if hostURI := mongoHostFromURI(uri); hostURI != "" {
		mc.appRuntime.Probe.AddHostURI(hostURI)
	}
	return mc
}

func (mc *MongoComponent) ReadPreference(rp *readpref.ReadPref) *MongoComponent {
	mc.readPreference = rp
	if mc.client != nil {
		mc.client.SetReadPreference(rp)
	}
	return mc
}

func (mc *MongoComponent) Timeout(timeout time.Duration) *MongoComponent {
	mc.timeout = timeout
	if mc.client != nil {
		mc.client.SetTimeout(timeout)
	}
	return mc
}

func (mc *MongoComponent) PoolSize(minSize, maxSize uint64) *MongoComponent {
	mc.minPoolSize = minSize
	mc.maxPoolSize = maxSize
	if mc.client != nil {
		mc.client.SetPoolSize(minSize, maxSize)
	}
	return mc
}

func (mc *MongoComponent) MaxConnecting(maxConnecting uint64) *MongoComponent {
	mc.maxConnecting = maxConnecting
	if mc.client != nil {
		mc.client.SetMaxConnecting(maxConnecting)
	}
	return mc
}

func (mc *MongoComponent) TLSConfig(conf *tls.Config) *MongoComponent {
	mc.tlsConfig = conf
	if mc.client != nil {
		mc.client.SetTLSConfig(conf)
	}
	return mc
}

func (mc *MongoComponent) SlowOperationThreshold(threshold time.Duration) *MongoComponent {
	mc.slowOperationThreshold = threshold
	return mc
}

func (mc *MongoComponent) LoadFromPrefix(prefix string) {
	if prefix == "" {
		panic("mongo property prefix is required")
	}

	mc.SetURI(mc.appRuntime.Property.RequiredProperty(prefix + ".uri"))

	switch mc.appRuntime.Property.Property(prefix + ".read.preference") {
	case "secondary":
		mc.ReadPreference(readpref.Secondary())
	case "secondaryPreferred":
		mc.ReadPreference(readpref.SecondaryPreferred())
	case "primaryPreferred":
		mc.ReadPreference(readpref.PrimaryPreferred())
	case "nearest":
		mc.ReadPreference(readpref.Nearest())
	}

	timeoutValue := mc.appRuntime.Property.Property(prefix + ".timeout")
	if timeoutValue != "" {
		timeout, err := time.ParseDuration(timeoutValue)
		if err != nil {
			panic(fmt.Sprintf("invalid mongo timeout for prefix %q: %v", prefix, err))
		}
		mc.Timeout(timeout)
	}

	minPoolValue := mc.appRuntime.Property.Property(prefix + ".pool.min")
	maxPoolValue := mc.appRuntime.Property.Property(prefix + ".pool.max")
	if minPoolValue != "" || maxPoolValue != "" {
		minPool, err := strconv.ParseUint(defaultString(minPoolValue, "0"), 10, 64)
		if err != nil {
			panic(fmt.Sprintf("invalid mongo min pool size for prefix %q: %v", prefix, err))
		}
		maxPool, err := strconv.ParseUint(defaultString(maxPoolValue, "0"), 10, 64)
		if err != nil {
			panic(fmt.Sprintf("invalid mongo max pool size for prefix %q: %v", prefix, err))
		}
		mc.PoolSize(minPool, maxPool)
	}

	maxConnectingValue := mc.appRuntime.Property.Property(prefix + ".max.connecting")
	if maxConnectingValue != "" {
		maxConnecting, err := strconv.ParseUint(maxConnectingValue, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("invalid mongo max connecting for prefix %q: %v", prefix, err))
		}
		mc.MaxConnecting(maxConnecting)
	}
}

func (mc *MongoComponent) ensureClient() *internalmongo.Client {
	if mc.client != nil {
		return mc.client
	}
	if mc.uri == "" {
		panic("mongo uri is required")
	}

	client := internalmongo.New()
	client.SetURI(mc.uri)
	if mc.readPreference != nil {
		client.SetReadPreference(mc.readPreference)
	}
	if mc.timeout > 0 {
		client.SetTimeout(mc.timeout)
	}
	if mc.minPoolSize > 0 || mc.maxPoolSize > 0 {
		client.SetPoolSize(mc.minPoolSize, mc.maxPoolSize)
	}
	if mc.maxConnecting > 0 {
		client.SetMaxConnecting(mc.maxConnecting)
	}
	if mc.tlsConfig != nil {
		client.SetTLSConfig(mc.tlsConfig)
	}

	mc.client = client
	return mc.client
}

func (mc *MongoComponent) ForceStart() {
	if err := mc.ensureClient().Initialize(context.Background()); err != nil {
		panic(fmt.Sprintf("mongo initialize failed: %v", err))
	}
}

func (mc *MongoComponent) Client() *orgmongo.Client {
	mc.ForceStart()
	return mc.client.Raw()
}

func (mc *MongoComponent) Ping(ctx context.Context) error {
	return mc.ensureClient().Ping(ctx)
}

func (mc *MongoComponent) MustPing(ctx context.Context) {
	if err := mc.Ping(ctx); err != nil {
		panic(fmt.Sprintf("mongo ping failed: %v", err))
	}
}

func (mc *MongoComponent) Close() error {
	if mc.client == nil {
		return nil
	}
	err := mc.client.Close()
	mc.client = nil
	return err
}

func mongoComponentName(name string) string {
	if name == "" {
		return defaultMongoComponentName
	}
	return defaultMongoComponentName + ":" + name
}

func mongoHostFromURI(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return ""
	}
	return u.Host
}
