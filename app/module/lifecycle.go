package module

import "github.com/austin72905/go-infra/internal/module"

type LifecycleManager struct {
	Startup  *module.StartupLifecycle
	Shutdown *module.ShutdownLifecycle
	Started  bool
}
