package app

import "fmt"

func (r *Runtime) RegisterComponent(name string, component Component) {
    if r.Lifecycle.Started {
        panic(fmt.Sprintf("%s component must be registered during Initialize()", name))
    }

    component.Initialize(r, name)
    component.Validate()
    r.Components.Register(name, component)
}
