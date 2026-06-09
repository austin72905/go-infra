package module

import (
	"context"
	"fmt"
	"net"

	internalgrpc "go-infra/internal/grpc"
	internalmodule "go-infra/internal/module"

	ogrpc "google.golang.org/grpc"
	"google.golang.org/grpc/health"
)

const grpcComponentName = "Grpc"

type GrpcComponent struct {
	appRuntime *AppRuntime

	server             *internalgrpc.Server
	listen             string
	maxConnections     int32
	opts               []ogrpc.ServerOption
	startRegistered    bool
	shutdownRegistered bool
}

func (gc *GrpcComponent) initialize(runtime *AppRuntime, name string) {
	gc.appRuntime = runtime

	if !gc.startRegistered {
		runtime.Lifecycle.Startup.AddServe(internalmodule.TaskFunc(func(ctx context.Context) {
			gc.Start(ctx)
		}))
		gc.startRegistered = true
	}

	if !gc.shutdownRegistered {
		runtime.Lifecycle.Shutdown.AddStopServer(internalmodule.TaskFunc(func(ctx context.Context) {
			_ = gc.Shutdown(ctx)
		}))
		gc.shutdownRegistered = true
	}
}

func (gc *GrpcComponent) validate() {
}

func (gc *GrpcComponent) AddOpt(opt ...ogrpc.ServerOption) *GrpcComponent {
	if gc.server != nil {
		panic("grpc server already configured, cannot add option")
	}
	gc.opts = append(gc.opts, opt...)
	return gc
}

func (gc *GrpcComponent) MaxConnections(max int32) *GrpcComponent {
	if gc.server != nil {
		panic("grpc server already configured, cannot set max connections")
	}
	gc.maxConnections = max
	return gc
}

func (gc *GrpcComponent) Listen(listen string) {
	gc.listen = listen
	if hostURI := grpcProbeHostURI(listen); hostURI != "" {
		gc.appRuntime.Probe.AddHostURI(hostURI)
	}
}

func (gc *GrpcComponent) LoadFromPrefix(prefix string) {
	if prefix == "" {
		panic("grpc property prefix is required")
	}

	gc.Listen(gc.appRuntime.Property.RequiredProperty(prefix + ".listen"))
}

func (gc *GrpcComponent) Server() *ogrpc.Server {
	return gc.ensureServer().GRPCServer()
}

func (gc *GrpcComponent) HealthServer() *health.Server {
	return gc.ensureServer().HealthServer()
}

func (gc *GrpcComponent) Start(ctx context.Context) {
	_ = ctx
	server := gc.ensureServer()
	if err := server.Start(); err != nil {
		panic(fmt.Sprintf("start grpc server failed: %v", err))
	}
}

func (gc *GrpcComponent) Shutdown(ctx context.Context) error {
	if gc.server == nil {
		return nil
	}
	return gc.server.Shutdown(ctx)
}

func (gc *GrpcComponent) ensureServer() *internalgrpc.Server {
	if gc.server != nil {
		return gc.server
	}
	if gc.listen == "" {
		panic("grpc listen is required")
	}

	opts := append([]ogrpc.ServerOption{}, gc.opts...)
	if gc.maxConnections > 0 {
		opts = append(opts, ogrpc.MaxConcurrentStreams(uint32(gc.maxConnections)))
	}

	server := internalgrpc.NewServer(opts...)
	server.SetListen(gc.listen)
	gc.server = server
	return gc.server
}

func grpcProbeHostURI(listen string) string {
	if listen == "" {
		return ""
	}

	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return ""
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		return ""
	}
	return net.JoinHostPort(host, port)
}
