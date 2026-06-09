package app

import (
	"context"

	internalmodule "github.com/austin72905/go-infra/internal/module"
)

func (r *Runtime) OnShutdown(fn func(ctx context.Context)) {
	if fn == nil {
		return
	}
	r.Lifecycle.Shutdown.AddReleaseResources(internalmodule.TaskFunc(fn))
}
