package app

type Component interface {
    initialize(runtime *Runtime, name string)
    validate()
}
