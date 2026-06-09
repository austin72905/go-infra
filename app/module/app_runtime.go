package module

import (
	"context"

	internalmodule "github.com/austin72905/go-infra/internal/module"
)

// Context 但是命名不佳
type AppRuntime struct {
	Lifecycle  *LifecycleManager
	Components *ComponentRegistry
	Web        *WebRuntime
	Property   *PropertyRuntime
	Probe      *StartupProbe
}

func NewAppRuntime() *AppRuntime {
	runtime := &AppRuntime{
		Lifecycle: &LifecycleManager{
			Startup:  &internalmodule.StartupLifecycle{},
			Shutdown: &internalmodule.ShutdownLifecycle{},
		},
		Components: NewComponentRegistry(),
		Web:        NewWebRuntime(),
		Property:   NewPropertyRuntime(),
		Probe:      NewStartupProbe(),
	}
	/*
		不是「只要跟生命週期有關就都隨便寫進去」，
		而是「這個 runtime 內的某個能力，如果需要在固定啟停階段被 framework 統一管理，就註冊進 lifecycle」
	*/
	runtime.Lifecycle.Startup.AddServe(internalmodule.TaskFunc(func(ctx context.Context) {
		// 當 app 進入 startup 的 Serve 階段時，幫我把 Web server 啟動。
		_ = runtime.Web.Start()
	}))
	runtime.Lifecycle.Shutdown.AddStopServer(internalmodule.TaskFunc(func(ctx context.Context) {
		// 當 app 進入 shutdown 的 StopServer 階段時，幫我把 Web server 關掉
		_ = runtime.Web.Shutdown(ctx)
	}))

	return runtime
}
