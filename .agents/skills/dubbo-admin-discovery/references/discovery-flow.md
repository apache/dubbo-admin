# Discovery Implementation Flow

This reference explains how registry-sourced discovery is implemented in dubbo-admin. Read it when changing `pkg/core/discovery/`, `pkg/core/controller/`, or discovery subscribers.

## End-to-end call chain

```text
runtime.RegisterComponent(newDiscoveryComponent)
  -> Bootstrap initializes discoveryComponent.Init
  -> Init gets EventBus and ResourceStore from BuilderContext
  -> Init loops ctx.Config().Discovery
  -> initInformers(cfg, storeRouter, eventBus)
  -> initSubscribes(storeRouter, eventBus, ctx.Config().Engine)
  -> Start
  -> startBusinessLogic
  -> EventBus.Subscribe(discovery subscribers)
  -> informer.Run(stopCh)
  -> cache.Controller ListAndWatch
  -> DeltaFIFO Process: informer.HandleDeltas
  -> store Add/Update/Delete
  -> informer.EmitEvent
  -> EventBus.Send
  -> subscriber.ProcessEvent
  -> derived resources are written and follow-up events may be emitted
```

## Component registration and dependencies

Key file: `pkg/core/discovery/component.go`

Discovery is registered as a runtime component:

```go
func init() {
    runtime.RegisterComponent(newDiscoveryComponent())
}
```

Its required dependencies are explicit:

```go
func (d *discoveryComponent) RequiredDependencies() []runtime.ComponentType {
    return []runtime.ComponentType{
        runtime.EventBus,
        runtime.ResourceStore,
    }
}
```

`Init` retrieves the activated EventBus and ResourceStore, then builds informers and subscribers:

```go
eventBusComponent, err := ctx.GetActivatedComponent(runtime.EventBus)
eventBus, ok := eventBusComponent.(events.EventBus)
d.subscriptionMgr = eventBus

storeComponent, err := ctx.GetActivatedComponent(runtime.ResourceStore)
storeRouter, ok := storeComponent.(store.Router)

d.configs = ctx.Config().Discovery
for _, cfg := range d.configs {
    informers, err := d.initInformers(cfg, storeRouter, eventBus)
    d.informers[cfg.ID] = informers
}

err = d.initSubscribes(storeRouter, eventBus, ctx.Config().Engine)
```

Memory store skips leader election. Non-memory stores may initialize leader election so only the leader runs discovery business logic.

## Factory and ListWatcher implementation

Key files:

- `pkg/core/discovery/factory.go`
- `pkg/core/controller/listwatcher.go`

The factory registry chooses an implementation by discovery type:

```go
type Factory interface {
    Support(discovery.Type) bool
    NewListWatchers(config *discovery.Config) ([]controller.ResourceListerWatcher, error)
}
```

Each discovery ListWatcher must expose the resource kind it produces:

```go
type ResourceListerWatcher interface {
    cache.ListerWatcher
    ResourceKind() coremodel.ResourceKind
    TransformFunc() cache.TransformFunc
}
```

`ResourceKind()` determines the backing store route used by the informer. `TransformFunc()` exists in the interface, but discovery's current `initInformers` does not call `informer.SetTransform(lw.TransformFunc())`; do not rely on discovery transforms without checking or changing that behavior.

## Informer creation

Key function: `(*discoveryComponent).initInformers`

```go
factory, err := ListWatcherFactoryRegistry().GetListWatcherFactory(cfg.Type)
lwList, err := factory.NewListWatchers(cfg)

for i, lw := range lwList {
    resourceStore, err := storeRouter.ResourceKindRoute(lw.ResourceKind())
    informer := controller.NewInformerWithOptions(
        lw,
        eventBus,
        resourceStore,
        keyFunc,
        controller.Options{ResyncPeriod: 0},
    )
    informers[i] = informer
}
```

Discovery uses `keyFunc` from `component.go`, which requires objects to implement `coremodel.Resource` and returns `r.ResourceKey()`.

## Start and leader election

Key functions:

- `(*discoveryComponent).Start`
- `(*discoveryComponent).startBusinessLogic`

If leader election is not needed, `Start` immediately calls `startBusinessLogic(ch)`. If leader election is needed, business logic runs only inside `onStartLeading`, using a leadership-specific stop channel. `onStopLeading` closes that channel so informer goroutines exit for the lost leadership term.

Business logic registers subscribers once and starts informers:

```go
if !d.subscribed.Load() {
    for _, sub := range d.subscribers {
        err := d.subscriptionMgr.Subscribe(sub)
    }
    d.subscribed.Store(true)
}

for name, informers := range d.informers {
    for _, informer := range informers {
        go informer.Run(stopCh)
    }
}
```

## Informer DeltaFIFO processing

Key file: `pkg/core/controller/informer.go`

`Run` builds a Kubernetes `DeltaFIFO` and `cache.Controller`:

```go
fifo := cache.NewDeltaFIFOWithOptions(cache.DeltaFIFOOptions{
    KnownObjects: s.indexer,
    EmitDeltaTypeReplaced: true,
    Transformer: s.transform,
    KeyFunction: s.keyFunc,
})

cfg := &cache.Config{
    Queue: fifo,
    ListerWatcher: s.listerWatcher,
    Process: s.HandleDeltas,
}

s.controller = cache.New(cfg)
s.controller.Run(stopCh)
```

`HandleDeltas` updates the store and emits normalized events:

```go
case cache.Sync, cache.Replaced, cache.Added, cache.Updated:
    if old, exists, err := s.indexer.Get(resource); err == nil && exists {
        s.indexer.Update(resource)
        s.EmitEvent(cache.Updated, old.(model.Resource), resource)
    } else {
        s.indexer.Add(resource)
        s.EmitEvent(cache.Added, nil, resource)
    }

case cache.Deleted:
    s.indexer.Delete(resource)
    s.EmitEvent(cache.Deleted, resource, nil)
```

Subscribers receive event types based on store existence, not only raw watch delta type. Delete handling supports `cache.DeletedFinalStateUnknown` tombstones in `toResource`.

## Event emission

Key function: `(*informer).EmitEvent`

```go
func (s *informer) EmitEvent(typ cache.DeltaType, oldObj model.Resource, newObj model.Resource) {
    event := events.NewResourceChangedEvent(typ, oldObj, newObj)
    s.emitter.Send(event)
}
```

Discovery relies on EventBus dispatch by resource kind. For delete events, `newObj` is nil and EventBus must route by `oldObj`.

## Subscriber registration

Key function: `(*discoveryComponent).initSubscribes`

Base subscribers are always registered:

```go
rpcInstanceSub := subscriber.NewRPCInstanceEventSubscriber(instanceStore, rtInstanceStore, emitter, engineConfig)
serviceConsumerMetadataSub := subscriber.NewServiceConsumerMetadataEventSubscriber(appStore, emitter)
serviceProviderMetadataSub := subscriber.NewServiceProviderMetadataEventSubscriber(
    appStore, serviceStore, serviceProviderMetadataStore, emitter)
instanceSub := subscriber.NewInstanceEventSubscriber(appStore, instanceStore, emitter)
d.subscribers = append(d.subscribers, rpcInstanceSub, serviceConsumerMetadataSub, serviceProviderMetadataSub, instanceSub)
```

Conditional subscribers:

```go
if hasNacosDiscovery {
    d.subscribers = append(d.subscribers, subscriber.NewNacosServiceEventSubscriber(emitter, storeRouter))
}

if hasZkDiscovery {
    d.subscribers = append(d.subscribers,
        subscriber.NewZKMetadataEventSubscriber(emitter, storeRouter),
        subscriber.NewZKConfigEventSubscriber(emitter, storeRouter))
}
```

## Provider metadata subscriber

Key file: `pkg/core/discovery/subscriber/service_provider_metadata.go`

Identity:

```go
func (s *ServiceProviderMetadataEventSubscriber) ResourceKind() coremodel.ResourceKind {
    return meshresource.ServiceProviderMetadataKind
}

func (s *ServiceProviderMetadataEventSubscriber) Name() string {
    return "Discovery-" + s.ResourceKind().ToString()
}
```

Event handling:

- Add, replace, sync: require `newObj`, call `processUpsert`.
- Update: require `newObj`, call `processUpdate`.
- Delete: require `oldObj`, call `processDelete`.

Upsert and update validate `Spec`, ensure the provider application exists, then sync the derived Service resource. Delete validates `Spec` and syncs the derived Service after provider metadata has already been removed from the store.

`ensureApplication` creates an `ApplicationResource` when `Spec.ProviderAppName` is present and missing, then emits an Application added event.

`syncService` derives a `ServiceResource` from all provider metadata with the same service identity:

```go
serviceKey := meshresource.BuildServiceIdentityKey(serviceName, version, group)
resources, err := s.providerStore.ListByIndexes([]index.IndexCondition{
    {IndexName: index.ByMeshIndex, Value: mesh, Operator: index.Equals},
    {IndexName: index.ByServiceProviderServiceKey, Value: serviceKey, Operator: index.Equals},
})
```

Behavior:

- No providers and no Service: do nothing.
- No providers and Service exists: delete Service and emit Service deleted event.
- Providers remain and Service does not exist: add Service and emit Service added event.
- Providers remain and Service exists: update Service and emit Service updated event.

`buildServiceSpec` aggregates unique method names, sorts them, and sets language from the first provider where language can be inferred. Language inference priority is explicit `parameters["language"]`, dubbo-go release prefix, then Java type hints in method signatures or exported type definitions.

## Consumer metadata subscriber

Key file: `pkg/core/discovery/subscriber/service_consumer_metadata.go`

Event handling:

- Add, update, replace, sync: require `newObj`, call `processUpsert`.
- Delete: ignored with a warning.

`processUpsert` creates an `ApplicationResource` for `Spec.ConsumerAppName` if it is non-blank and missing, then emits an Application added event.

## Common failure modes

- Missing EventBus or ResourceStore activated component means runtime dependency setup is wrong.
- Unsupported discovery type means no factory registered with `Support(type) == true`.
- Missing store route for a ListWatcher resource kind prevents informer creation.
- Subscriber type assertions fail if informer emits the wrong resource type.
- Delete logic fails when old object is nil.
- Provider Service derivation depends on `ByMeshIndex` and `ByServiceProviderServiceKey`.

## Review checklist

- Confirm the ListWatcher `ResourceKind()` matches the store route and subscriber `ResourceKind()`.
- Confirm informer key function matches transformed resource keys if transforms are introduced.
- Confirm event type semantics after `HandleDeltas`, not just raw watch delta semantics.
- Confirm follow-up events are emitted when subscribers create, update, or delete derived resources.
- Confirm store indexes exist for every `ListByIndexes` query used by subscriber logic.
