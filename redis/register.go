package redis

import (
	"fmt"

	"github.com/austin72905/go-infra/app"
)

func Register(runtime *app.Runtime, name string) *Component {
	componentName := ComponentName(name)
	if component, ok := runtime.Components.LookupComponent(componentName); ok {
		cache, typeOK := component.(*Component)
		if !typeOK {
			panic(fmt.Sprintf("component %q is not a redis.Component", componentName))
		}
		return cache
	}

	cache := &Component{}
	runtime.RegisterComponent(componentName, cache)
	return cache
}
