# Runtime Component Contracts

This reference explains the component abstraction used by dubbo-admin runtime. Read it before adding a component, changing component interfaces, or modifying dependency declarations.

## Key files

- `pkg/core/runtime/component.go`
- `pkg/core/runtime/registry.go`
- `pkg/core/runtime/builder.go`

## Component contract

`pkg/core/runtime/component.go` defines the runtime abstraction:

```go
type Lifecycle interface {
    Init(ctx BuilderContext) error
    Start(Runtime, <-chan struct{}) error
}

type Attribute interface {
    Type() ComponentType
    Order() int
    RequiredDependencies() []ComponentType
}

type Component interface {
    Attribute
    Lifecycle
}
```

Meaning:

- `Type()` is the stable identity used by registry, builder, runtime lookup, and dependency graph.
- `Init(ctx BuilderContext)` runs during bootstrap before the component is activated.
- `Start(Runtime, stop)` runs after runtime is built.
- `RequiredDependencies()` controls initialization order.
- `Order()` is deprecated for dependency management but still controls runtime start ordering.

## Built-in component types

```go
const (
    Console           ComponentType = "console"
    ResourceManager   ComponentType = "resource manager"
    ResourceStore     ComponentType = "resource store"
    ResourceEngine    ComponentType = "resource engine"
    ResourceDiscovery ComponentType = "resource discovery"
    EventBus          ComponentType = "event bus"
    RuleGovernor      ComponentType = "rule governor"
)
```

`CoreComponentTypes` contains `Console`, `ResourceManager`, `ResourceStore`, `ResourceEngine`, `ResourceDiscovery`, and `RuleGovernor`. Runtime start panics when a core component returns an error.

## Registry implementation

Key functions in `pkg/core/runtime/registry.go`:

```go
var registry = NewRegistry()

func ComponentRegistry() Registry {
    return registry
}

func RegisterComponent(component Component) {
    if err := registry.Register(component); err != nil {
        panic(err)
    }
}
```

The registry is global. Packages register components from `init()` functions, for example discovery and engine do this. Duplicate component types fail fast:

```go
func (r *componentRegistry) Register(component Component) error {
    _, ok := r.directory[component.Type()]
    if ok {
        return componentAlreadyRegisteredError(component.Type())
    }
    r.directory[component.Type()] = component
    return nil
}
```

Lookup is by component type:

```go
func (r *componentRegistry) Get(typ ComponentType) (Component, error) {
    component, ok := r.directory[typ]
    if !ok {
        return nil, noSuchComponentError(typ)
    }
    return component, nil
}
```

## Builder context contract

`pkg/core/runtime/builder.go` defines the context available during `Init`:

```go
type BuilderContext interface {
    Config() app.AdminConfig
    GetActivatedComponent(typ ComponentType) (Component, error)
    ActivateComponent(comp Component) error
}
```

A component can only retrieve dependencies that have already been activated by bootstrap. This is why `RequiredDependencies()` must be correct.

## Implementation example

A component that needs EventBus and ResourceStore should declare both:

```go
func (c *myComponent) RequiredDependencies() []runtime.ComponentType {
    return []runtime.ComponentType{runtime.EventBus, runtime.ResourceStore}
}
```

During `Init`, it should retrieve dependencies from `BuilderContext`:

```go
eventBusComponent, err := ctx.GetActivatedComponent(runtime.EventBus)
storeComponent, err := ctx.GetActivatedComponent(runtime.ResourceStore)
```

If the dependency is missing here, the problem is usually dependency declaration or bootstrap ordering, not runtime start order.

## Review checks

- New component type is stable and unique.
- New component is registered exactly once.
- `RequiredDependencies()` lists every component used via `GetActivatedComponent` during `Init`.
- `Order()` is not used to compensate for missing `RequiredDependencies()`.
- Core failure semantics are considered if a new component should be fatal during runtime start.
