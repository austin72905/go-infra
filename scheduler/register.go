package scheduler

import (
	"fmt"

	"github.com/austin72905/go-infra/app"
)

func Register(runtime *app.Runtime, name string) *Component {
	componentName := ComponentName(name)
	if component, ok := runtime.Components.LookupComponent(componentName); ok {
		scheduler, typeOK := component.(*Component)
		if !typeOK {
			panic(fmt.Sprintf("component %q is not a scheduler.Component", componentName))
		}
		return scheduler
	}

	scheduler := &Component{}
	runtime.RegisterComponent(componentName, scheduler)
	return scheduler
}

func ComponentName(name string) string {
	if name == "" {
		return DefaultComponentName
	}
	return DefaultComponentName + ":" + name
}
