package postgres

import (
	"fmt"

	"github.com/austin72905/go-infra/app"
)

func Register(runtime *app.Runtime, name string) *Component {
	componentName := ComponentName(name)
	if component, ok := runtime.Components.LookupComponent(componentName); ok {
		database, typeOK := component.(*Component)
		if !typeOK {
			panic(fmt.Sprintf("component %q is not a postgres.Component", componentName))
		}
		return database
	}

	database := &Component{}
	runtime.RegisterComponent(componentName, database)
	return database
}
