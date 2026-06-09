package dedupe

import (
	"fmt"

	"github.com/austin72905/go-infra/app"
	"github.com/austin72905/go-infra/redis"
)

func Register(runtime *app.Runtime, name string) *Component {
	componentName := ComponentName(name)
	if component, ok := runtime.Components.LookupComponent(componentName); ok {
		deduplicator, typeOK := component.(*Component)
		if !typeOK {
			panic(fmt.Sprintf("component %q is not a dedupe.Component", componentName))
		}
		return deduplicator
	}

	_ = redis.Register(runtime, name)
	deduplicator := &Component{redisComponentName: RedisComponentName(name)}
	runtime.RegisterComponent(componentName, deduplicator)
	return deduplicator
}
