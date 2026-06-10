package scheduler

import (
	"context"
	"time"

	"github.com/austin72905/go-infra/app"
	internalmodule "github.com/austin72905/go-infra/internal/module"
	internalscheduler "github.com/austin72905/go-infra/internal/scheduler"
)

const DefaultComponentName = "Scheduler"

type JobID = internalscheduler.JobID

type JobInfo = internalscheduler.JobInfo

type Component struct {
	appRuntime         *app.Runtime
	engine             *internalscheduler.Engine
	location           *time.Location
	panicOnAnyAddError bool
	startupRegistered  bool
	shutdownRegistered bool
}

func (sc *Component) Initialize(runtime *app.Runtime, name string) {
	sc.appRuntime = runtime

	if !sc.startupRegistered {
		runtime.Lifecycle.Startup.AddStart(internalmodule.TaskFunc(func(ctx context.Context) {
			sc.Start()
		}))
		sc.startupRegistered = true
	}

	if !sc.shutdownRegistered {
		runtime.Lifecycle.Shutdown.AddStopBackground(internalmodule.TaskFunc(func(ctx context.Context) {
			sc.Stop()
		}))
		runtime.Lifecycle.Shutdown.AddAwaitBackground(internalmodule.TaskFunc(func(ctx context.Context) {
			sc.AwaitTermination(ctx)
		}))
		sc.shutdownRegistered = true
	}
}

func (sc *Component) Validate() {}

func (sc *Component) SetPanicOnAnyAddError(val bool) {
	sc.panicOnAnyAddError = val
	if sc.engine != nil {
		sc.engine.PanicOnAnyAddError(val)
	}
}

func (sc *Component) SetLocation(location time.Location) {
	sc.location = &location
}

func (sc *Component) AddFuncJob(spec string, fn func(context.Context)) (JobID, error) {
	sc.ensureEngine()
	return sc.engine.AddFunc(spec, fn)
}

func (sc *Component) AddFuncJobWithName(spec, name string, fn func(context.Context)) (JobID, error) {
	sc.ensureEngine()
	return sc.engine.AddFuncWithName(spec, name, fn)
}

func (sc *Component) Remove(id JobID) {
	sc.ensureEngine()
	sc.engine.Remove(id)
}

func (sc *Component) Start() {
	sc.ensureEngine()
	sc.engine.Start()
}

func (sc *Component) Stop() {
	if sc.engine == nil {
		return
	}
	sc.engine.Stop()
}

func (sc *Component) AwaitTermination(ctx context.Context) {
	if sc.engine == nil {
		return
	}
	sc.engine.AwaitTermination(ctx)
}

func (sc *Component) JobsInfo() []JobInfo {
	sc.ensureEngine()
	return sc.engine.JobsInfo()
}

func (sc *Component) RunningTasks() int {
	sc.ensureEngine()
	return sc.engine.RunningTasks()
}

func (sc *Component) ensureEngine() {
	if sc.engine != nil {
		return
	}

	sc.engine = internalscheduler.New(sc.location)
	sc.engine.PanicOnAnyAddError(sc.panicOnAnyAddError)
}
