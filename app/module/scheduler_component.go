package module

import (
	"context"
	internalmodule "go-infra/internal/module"
	"go-infra/internal/scheduler"
	"time"
)

const schedulerComponentName = "Scheduler"

type SchedulerComponent struct {
	appRuntime          *AppRuntime
	engine              *scheduler.Engine
	location            *time.Location
	startTaskRegistered bool // 是否註冊過，避免重複註冊
	stopTaskRegistered  bool
}

func (sc *SchedulerComponent) initialize(runtime *AppRuntime, name string) {
	sc.appRuntime = runtime
	if sc.startTaskRegistered {
		return
	}
	// SchedulerComponent 必須在 Initialize() 期間先建立，之後由 StartupLifecycle.Start 統一啟動。
	runtime.Lifecycle.Startup.AddStart(internalmodule.TaskFunc(func(ctx context.Context) {
		sc.Start()
	}))
	sc.startTaskRegistered = true

	if sc.stopTaskRegistered {
		return
	}
	runtime.Lifecycle.Shutdown.AddStopBackground(internalmodule.TaskFunc(func(ctx context.Context) {
		sc.Stop() // 先停止新的 cron 觸發
	}))
	runtime.Lifecycle.Shutdown.AddAwaitBackground(internalmodule.TaskFunc(func(ctx context.Context) {
		sc.AwaitTermination(ctx) //等還在跑的 job 收完
	}))
	sc.stopTaskRegistered = true
}

func (sc *SchedulerComponent) validate() {

}

func (sc *SchedulerComponent) ensureEngine() {
	if sc.engine != nil {
		return
	}

	sc.engine = scheduler.New(sc.location)
}

// 新增 job 時如果 spec 錯了或 add 失敗 要不要直接 panic
func (sc *SchedulerComponent) SetPanicOnAnyAddError(val bool) {
	sc.ensureEngine()
	sc.engine.PanicOnAnyAddError(val)
}

// 設置時區
func (sc *SchedulerComponent) SetLocation(location time.Location) {
	sc.location = &location
}

// spec = 執行頻率
func (sc *SchedulerComponent) AddFuncJob(spec string, fn func(context.Context)) (scheduler.JobID, error) {
	sc.ensureEngine()
	return sc.engine.AddFunc(spec, fn)
}

func (sc *SchedulerComponent) AddFuncJobWithName(spec, name string, fn func(ctx context.Context)) (scheduler.JobID, error) {
	sc.ensureEngine()
	return sc.engine.AddFuncWithName(spec, name, fn)
}

// 手動刪除job
func (sc *SchedulerComponent) Remove(id scheduler.JobID) {
	sc.ensureEngine()
	sc.engine.Remove(id)
}

func (sc *SchedulerComponent) Start() {
	sc.ensureEngine()
	sc.engine.Start()
}

func (sc *SchedulerComponent) Stop() {
	if sc.engine == nil {
		return
	}
	sc.engine.Stop()
}

func (sc *SchedulerComponent) AwaitTermination(ctx context.Context) {
	if sc.engine == nil {
		return
	}
	sc.engine.AwaitTermination(ctx)
}

func (sc *SchedulerComponent) JobsInfo() []scheduler.JobInfo {
	sc.ensureEngine()
	return sc.engine.JobsInfo()
}

func (sc *SchedulerComponent) RunningTasks() int {
	sc.ensureEngine()
	return sc.engine.RunningTasks()
}
