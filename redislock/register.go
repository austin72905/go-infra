package redislock

import (
	"fmt"

	"github.com/austin72905/go-infra/app"
	"github.com/austin72905/go-infra/redis"
)

func Register(runtime *app.Runtime, name string) *Component {
	componentName := ComponentName(name)
	if component, ok := runtime.Components.LookupComponent(componentName); ok {
		locker, typeOK := component.(*Component)
		if !typeOK {
			panic(fmt.Sprintf("component %q is not a redislock.Component", componentName))
		}
		return locker
	}

	_ = redis.Register(runtime, name)
	locker := &Component{redisComponentName: RedisComponentName(name)}
	runtime.RegisterComponent(componentName, locker)
	return locker
}
