package app

import (
    "context"
    "os"
    "os/signal"
    "syscall"
)

type Module interface {
    SetAppRuntime(appRuntime *Runtime)
    Initialize()
}

type AppInterface interface {
    Module
    Start()
    SetupRuntime()
}

func Start(app AppInterface) {
    app.SetupRuntime()
    app.Initialize()
    app.Start()
}

type App struct {
    RuntimeHolder
}

func (app *App) Start() {
    ctx := context.Background()
    if err := app.Runtime.Property.Validate(); err != nil {
        panic(err)
    }
    if err := app.Runtime.Probe.Check(ctx); err != nil {
        panic(err)
    }
    app.Runtime.Lifecycle.Startup.RunAll(ctx)
    app.Runtime.Lifecycle.Started = true
    app.waitForShutdownSignal()
    app.shutdown(ctx)
}

func (app *App) SetupRuntime() {
    runtime := NewRuntime()
    app.SetAppRuntime(runtime)
}

func (app *App) waitForShutdownSignal() {
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
    defer signal.Stop(stop)

    <-stop
}

func (app *App) shutdown(ctx context.Context) {
    app.Runtime.Lifecycle.Shutdown.RunAll(ctx)
}
