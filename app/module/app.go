package module

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

type Module interface {
	SetAppRuntime(appRuntime *AppRuntime) // SetContext. framework內部實作
	Initialize()                          // 各服務一定要實作，才能實現介面,app.Schedule一定要在這裡呼叫
}

// 介面
type AppInterface interface {
	Module
	Start()
	SetupRuntime()
}

// 讓各服務呼叫 , 啟動傳入的實作類
func StartServer(app AppInterface) {
	app.SetupRuntime()
	app.Initialize()
	app.Start()
}

// 實作類
type App struct {
	ModuleBase
}

// *App 有實作 AppInterface  ; 但是 App 沒有
func (app *App) Start() {
	ctx := context.Background()
	if err := app.AppRuntime.Property.Validate(); err != nil {
		panic(err)
	}
	if err := app.AppRuntime.Probe.Check(ctx); err != nil {
		panic(err)
	}
	app.AppRuntime.Lifecycle.Startup.RunAll(ctx)
	app.AppRuntime.Lifecycle.Started = true
	app.waitForShutdownSignal()
	app.shutdown(ctx)
}

// 設置runtime 管理多組建的配置,關閉順序...
func (app *App) SetupRuntime() {

	runtime := NewAppRuntime()
	app.SetAppRuntime(runtime)
}

// 等待接收 SIGTERM / Ctrl+C 訊號
func (app *App) waitForShutdownSignal() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	<-stop
}

func (app *App) shutdown(ctx context.Context) {
	app.AppRuntime.Lifecycle.Shutdown.RunAll(ctx)
}
