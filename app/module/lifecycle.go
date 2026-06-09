package module

import "go-infra/internal/module"

type LifecycleManager struct {
	Startup  *module.StartupLifecycle
	Shutdown *module.ShutdownLifecycle
	Started  bool
}
