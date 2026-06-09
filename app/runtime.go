package app

import (
    "context"

    "github.com/austin72905/go-infra/config"
    internalmodule "github.com/austin72905/go-infra/internal/module"
    "github.com/austin72905/go-infra/probe"
    "github.com/austin72905/go-infra/web"
)

type Runtime struct {
    Lifecycle  *LifecycleManager
    Components *ComponentRegistry
    Web        *web.Runtime
    Property   *config.PropertyRuntime
    Probe      *probe.StartupProbe
}

func NewRuntime() *Runtime {
    runtime := &Runtime{
        Lifecycle: &LifecycleManager{
            Startup:  &internalmodule.StartupLifecycle{},
            Shutdown: &internalmodule.ShutdownLifecycle{},
        },
        Components: NewComponentRegistry(),
        Web:        web.New(),
        Property:   config.NewPropertyRuntime(),
        Probe:      probe.NewStartupProbe(),
    }

    runtime.Lifecycle.Startup.AddServe(internalmodule.TaskFunc(func(ctx context.Context) {
        _ = runtime.Web.Start()
    }))
    runtime.Lifecycle.Shutdown.AddStopServer(internalmodule.TaskFunc(func(ctx context.Context) {
        _ = runtime.Web.Shutdown(ctx)
    }))

    return runtime
}

type RuntimeHolder struct {
    Runtime *Runtime
}

func (h *RuntimeHolder) SetAppRuntime(appRuntime *Runtime) {
    h.Runtime = appRuntime
}

func (h *RuntimeHolder) Web() *web.Runtime {
    return h.Runtime.Web
}

func (h *RuntimeHolder) PropertyRuntime() *config.PropertyRuntime {
    return h.Runtime.Property
}

func (h *RuntimeHolder) Probe() *probe.StartupProbe {
    return h.Runtime.Probe
}
