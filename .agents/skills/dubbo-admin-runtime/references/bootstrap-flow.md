# Runtime Bootstrap Flow

This reference explains how dubbo-admin initializes components before building the runtime.

## End-to-end call chain

```text
bootstrap.Bootstrap(appCtx, cfg)
  -> runtime.BuilderFor(appCtx, cfg)
  -> NewSmartBootstrapper(builder)
  -> bootstrapper.bootstrapComponents(appCtx, cfg)
  -> gatherComponents()
  -> sortComponents(components)
  -> runtime.NewDependencyGraph(components).TopologicalSort()
  -> initAndActivateComponent(builder, comp)
  -> comp.Init(builder)
  -> builder.ActivateComponent(comp)
  -> builder.Build()
  -> runtime.Start(stop)
```

## Bootstrap entrypoint

Key file: `pkg/core/bootstrap/bootstrap.go`

```go
func Bootstrap(appCtx context.Context, cfg app.AdminConfig) (runtime.Runtime, error) {
    builder, err := runtime.BuilderFor(appCtx, cfg)
    bootstrapper := NewSmartBootstrapper(builder)
    if err := bootstrapper.bootstrapComponents(appCtx, cfg); err != nil {
        return nil, err
    }
    rt, err := builder.Build()
    return rt, nil
}
```

`BuilderFor` creates runtime identity and an initially empty activated component map:

```go
return &Builder{
    cfg: cfg,
    appCtx: appCtx,
    runtimeInfo: runtimeInfo{
        instanceId: fmt.Sprintf("%s-%s", hostname, suffix),
        clusterId:  fmt.Sprintf("%s-%s", hostname, suffix),
        startTime:  time.Now(),
    },
    components: make(map[ComponentType]Component),
}, nil
```

## Component gathering

`gatherComponents` loads required core components from the global registry:

```go
coreComps := []struct {
    name   string
    getter func() (runtime.Component, error)
}{
    {"EventBus", runtime.ComponentRegistry().EventBus},
    {"ResourceStore", runtime.ComponentRegistry().ResourceStore},
    {"ResourceDiscovery", runtime.ComponentRegistry().ResourceDiscovery},
    {"ResourceEngine", runtime.ComponentRegistry().ResourceEngine},
    {"ResourceManager", runtime.ComponentRegistry().ResourceManager},
    {"Console", runtime.ComponentRegistry().Console},
    {"RuleGovernor", runtime.ComponentRegistry().RuleGovernor},
}
```

If a core component is missing, bootstrap fails. Optional components are checked with `ComponentRegistry().Get`; missing optional components are logged and skipped:

- CounterManager
- DiagnosticsServer
- DistributedLock

## Sorting and initialization

`sortComponents` delegates to the dependency graph:

```go
graph := runtime.NewDependencyGraph(components)
sorted, err := graph.TopologicalSort()
```

`bootstrapComponents` initializes in sorted order:

```go
for i, comp := range ordered {
    if err := initAndActivateComponent(sb.builder, comp); err != nil {
        return bizerror.Wrap(err, bizerror.UnknownError,
            fmt.Sprintf("failed to initialize component %s", comp.Type()))
    }
}
```

`initAndActivateComponent` is the critical invariant:

```go
if err := comp.Init(builder); err != nil {
    return err
}
if err := builder.ActivateComponent(comp); err != nil {
    return bizerror.Wrap(err, bizerror.UnknownError, fmt.Sprintf("failed to activate %s", comp.Type()))
}
```

A component is visible to later components only after successful `Init` and activation.

## Build validation

`Builder.Build` requires all core components to be activated:

```go
for _, typ := range CoreComponentTypes {
    if _, exists := b.components[typ]; !exists {
        return nil, errors.Errorf("%v has not been configured", typ)
    }
}
```

It then returns a `runtime` containing runtime info, config, app context, and activated components.

## Review checks

- If `GetActivatedComponent` fails inside `Init`, inspect `RequiredDependencies()` and dependency graph order.
- If bootstrap fails while gathering, inspect package imports and `init()` registration.
- Optional components should not be added to `CoreComponentTypes` unless absence should be fatal.
- Do not activate a component before successful `Init`.
