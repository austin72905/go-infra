package app

type Component interface {
    Initialize(runtime *Runtime, name string)
    Validate()
}
