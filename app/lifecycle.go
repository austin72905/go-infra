package app

import internalmodule "github.com/austin72905/go-infra/internal/module"

type LifecycleManager struct {
    Startup  *internalmodule.StartupLifecycle
    Shutdown *internalmodule.ShutdownLifecycle
    Started  bool
}
