package rabbitmq

import (
	"fmt"

	"github.com/austin72905/go-infra/app"
)

func Register(runtime *app.Runtime, name string) *Component {
	componentName := ComponentName(name)
	if component, ok := runtime.Components.LookupComponent(componentName); ok {
		mq, typeOK := component.(*Component)
		if !typeOK {
			panic(fmt.Sprintf("component %q is not a rabbitmq.Component", componentName))
		}
		return mq
	}

	mq := &Component{}
	runtime.RegisterComponent(componentName, mq)
	return mq
}
