package module

import (
	"context"
	"fmt"
	"net"

	internalgrpc "go-infra/internal/grpc"
	internalmodule "go-infra/internal/module"

	ogrpc "google.golang.org/grpc"
)

const grpcClientComponentName = "GrpcClient"

type GrpcClientComponent struct {
	appRuntime *AppRuntime

	poolSize           int
	globalOpts         []ogrpc.DialOption
	pools              map[string]*internalgrpc.ClientPool
	registered         map[string]string
	shutdownRegistered bool
}

func (gc *GrpcClientComponent) initialize(runtime *AppRuntime, name string) {
	gc.appRuntime = runtime
	if gc.pools == nil {
		gc.pools = make(map[string]*internalgrpc.ClientPool)
	}
	if gc.registered == nil {
		gc.registered = make(map[string]string)
	}
	if gc.poolSize <= 0 {
		gc.poolSize = internalgrpc.DefaultConnPoolSize
	}
	if gc.shutdownRegistered {
		return
	}

	runtime.Lifecycle.Shutdown.AddReleaseResources(internalmodule.TaskFunc(func(ctx context.Context) {
		_ = gc.Close()
	}))
	gc.shutdownRegistered = true
}

func (gc *GrpcClientComponent) validate() {
}

func (gc *GrpcClientComponent) SetPoolSize(poolSize int) *GrpcClientComponent {
	if len(gc.pools) > 0 {
		panic("grpc clients already registered, cannot change pool size")
	}
	if poolSize <= 0 {
		poolSize = 1
	}
	gc.poolSize = poolSize
	return gc
}

func (gc *GrpcClientComponent) AddOpt(opt ...ogrpc.DialOption) *GrpcClientComponent {
	if len(gc.pools) > 0 {
		panic("grpc clients already registered, cannot add dial options")
	}
	gc.globalOpts = append(gc.globalOpts, opt...)
	return gc
}

func (gc *GrpcClientComponent) Register(serviceName, address string, opt ...ogrpc.DialOption) {
	if serviceName == "" {
		panic("grpc client service name is required")
	}
	if address == "" {
		panic(fmt.Sprintf("grpc client address is required for service %q", serviceName))
	}
	if _, ok := gc.pools[serviceName]; ok {
		panic(fmt.Sprintf("grpc client service %q already registered", serviceName))
	}

	if hostURI := grpcClientProbeHostURI(address); hostURI != "" {
		gc.appRuntime.Probe.AddHostURI(hostURI)
	}

	dialOpts := append([]ogrpc.DialOption{}, gc.globalOpts...)
	dialOpts = append(dialOpts, opt...)
	gc.pools[serviceName] = internalgrpc.NewClientPool(address, gc.poolSize, dialOpts...)
	gc.registered[serviceName] = address
}

func (gc *GrpcClientComponent) RegisterByProperty(serviceName, propertyKey string, opt ...ogrpc.DialOption) {
	if propertyKey == "" {
		panic("grpc client property key is required")
	}
	gc.Register(serviceName, gc.appRuntime.Property.RequiredProperty(propertyKey), opt...)
}

func (gc *GrpcClientComponent) Conn(serviceName string) (*ogrpc.ClientConn, error) {
	pool, ok := gc.pools[serviceName]
	if !ok {
		return nil, fmt.Errorf("unregistered grpc client service %q", serviceName)
	}
	return pool.Get(), nil
}

func (gc *GrpcClientComponent) MustConn(serviceName string) *ogrpc.ClientConn {
	conn, err := gc.Conn(serviceName)
	if err != nil {
		panic(err)
	}
	return conn
}

func (gc *GrpcClientComponent) Close() error {
	var firstErr error
	for name, pool := range gc.pools {
		if pool == nil {
			continue
		}
		if err := pool.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("close grpc client pool %q: %w", name, err)
		}
	}
	return firstErr
}

func grpcClientProbeHostURI(address string) string {
	if address == "" {
		return ""
	}

	host, port, err := net.SplitHostPort(address)
	if err == nil {
		if host == "" || host == "0.0.0.0" || host == "::" {
			return ""
		}
		return net.JoinHostPort(host, port)
	}

	return address
}

func NewGrpcClient[T any](component *GrpcClientComponent, serviceName string, factory func(ogrpc.ClientConnInterface) T) T {
	return factory(component.MustConn(serviceName))
}
