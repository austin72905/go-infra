package app

type ComponentRegistry struct {
    components map[string]Component
}

func NewComponentRegistry() *ComponentRegistry {
    return &ComponentRegistry{components: make(map[string]Component)}
}

func (cr *ComponentRegistry) Register(name string, component Component) {
    cr.components[name] = component
}

func (cr *ComponentRegistry) LookupComponent(name string) (Component, bool) {
    component, ok := cr.components[name]
    return component, ok
}

func (cr *ComponentRegistry) GetComponent(name string) Component {
    return cr.components[name]
}
