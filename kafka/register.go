//go:build cgo

package kafka

import (
	"fmt"

	"github.com/austin72905/go-infra/app"
)

func Register(runtime *app.Runtime, name string) *Component {
	componentName := ComponentName(name)
	if component, ok := runtime.Components.LookupComponent(componentName); ok {
		stream, typeOK := component.(*Component)
		if !typeOK {
			panic(fmt.Sprintf("component %q is not a kafka.Component", componentName))
		}
		return stream
	}

	stream := &Component{}
	runtime.RegisterComponent(componentName, stream)
	return stream
}
