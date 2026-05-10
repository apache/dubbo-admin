# Runtime Start Behavior

This reference explains what happens after bootstrap builds a runtime and `Start(stop)` is called.

## Key file

- `pkg/core/runtime/runtime.go`

## Runtime shape

`Runtime` combines runtime metadata, runtime context, and component management:

```go
type Runtime interface {
    RuntimeInfo
    RuntimeContext
    ComponentManager
}
```

`runtimeContext` stores config, app context, and activated components. `GetComponent` looks up an activated component by type after runtime build.

## Start call chain

```text
runtime.Start(stop)
  -> collect activated components from rt.components
  -> sort by descending Order()
  -> for each component, launch goroutine
  -> component.Start(rt, stop)
  -> panic on core component error, log on non-core error
  -> wait until stop channel closes
```

## Key implementation

```go
func (rt *runtime) Start(stop <-chan struct{}) error {
    components := maputil.Values(rt.components)
    slice.SortBy(components, func(a, b Component) bool {
        return a.Order() > b.Order()
    })
    for _, com := range components {
        go func() {
            err := com.Start(rt, stop)
            if err != nil {
                if slice.Contain(CoreComponentTypes, com.Type()) {
                    panic("core component " + com.Type() + " running failed with error: " + err.Error())
                } else {
                    logger.Errorf("component %s running failed with error: %s", com.Type(), err.Error())
                }
            } else {
                logger.Infof("component %s started successfully", com.Type())
            }
        }()
    }
    logger.Info("Admin started successfully")
    select {
    case <-stop:
        return nil
    }
}
```

## Important distinction

`RequiredDependencies()` controls bootstrap initialization order. `Order()` controls runtime start order only. If a component needs another component during `Init`, declare it in `RequiredDependencies()`.

## Concurrency caveat

The goroutine closes over the loop variable `com`. If changing this code, consider capturing the loop variable explicitly to avoid accidental closure bugs in Go versions or refactors where loop variable semantics matter:

```go
for _, com := range components {
    com := com
    go func() { _ = com.Start(rt, stop) }()
}
```

Do not change this casually without tests, because startup failure behavior is important for core components.

## Review checks

- New `Start` implementations should respect the stop channel.
- Long-running work belongs in `Start`; runtime already starts components in goroutines.
- Core component failure should remain fatal unless product behavior intentionally changes.
- Non-core component failure should not stop the whole runtime unless promoted to core semantics.
- Startup ordering must not be used as a substitute for initialization dependencies.
