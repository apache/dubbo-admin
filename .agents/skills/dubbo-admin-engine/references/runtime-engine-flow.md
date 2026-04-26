# Resource Engine Implementation Flow

This reference explains how dubbo-admin watches runtime infrastructure, converts it to RuntimeInstance resources, and merges runtime state into Instance resources.

## End-to-end call chain

```text
runtime.RegisterComponent(newEngineComponent)
  -> Bootstrap initializes engineComponent.Init
  -> Init gets EventBus and ResourceStore
  -> initInformers(cfg, eventBus)
  -> FactoryRegistry().GetListWatcherFactory(cfg.Type)
  -> factory.NewListWatchers(cfg)
  -> controller.NewInformerWithOptions(lw, eventBus, store, resolveInformerKeyFunc(lw), ...)
  -> initSubscribers(eventBus)
  -> Start
  -> startBusinessLogic or leader-gated startBusinessLogic
  -> EventBus.Subscribe(RuntimeInstanceEventSubscriber)
  -> informer.Run(stopCh)
  -> RuntimeInstance events
  -> RuntimeInstanceEventSubscriber.ProcessEvent
  -> Instance resource created, updated, or deleted
  -> follow-up Instance event emitted
```

## Engine component

Key file: `pkg/core/engine/component.go`

The engine component registers itself as `runtime.ResourceEngine` and depends on EventBus and ResourceStore:

```go
func (e *engineComponent) RequiredDependencies() []runtime.ComponentType {
    return []runtime.ComponentType{runtime.EventBus, runtime.ResourceStore}
}
```

During `Init`, it loads engine config, gets EventBus, casts ResourceStore to `store.Router`, initializes informers, initializes subscribers, and may configure leader election for non-memory stores.

```go
cfg := ctx.Config().Engine
e.name = cfg.ID

eventBusComponent, err := ctx.GetActivatedComponent(runtime.EventBus)
eventBus, ok := eventBusComponent.(events.EventBus)
e.subscriptionManager = eventBus

storeComponent, err := ctx.GetActivatedComponent(runtime.ResourceStore)
storeRouter, ok := storeComponent.(store.Router)
e.storeRouter = storeRouter

e.initInformers(cfg, eventBus)
e.initSubscribers(eventBus)
```

## Factory and Kubernetes ListWatcher

Key files:

- `pkg/core/engine/factory.go`
- `pkg/engine/kubernetes/factory.go`
- `pkg/engine/kubernetes/listerwatcher/runtime_instance.go`

Engine factories create `controller.ResourceListerWatcher` instances from engine config:

```go
type Factory interface {
    Support(enginecfg.Type) bool
    NewListWatchers(*enginecfg.Config) ([]controller.ResourceListerWatcher, error)
}
```

Kubernetes factory supports `enginecfg.Kubernetes`, builds either kubeconfig or in-cluster config, creates a Kubernetes clientset, then creates a Pod ListWatcher:

```go
if !strutil.IsBlank(kubeconfigPath) {
    config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
} else {
    config, err = rest.InClusterConfig()
}
clientset, err := kubernetes.NewForConfig(config)
podListerWatcher, err := listerwatcher.NewPodListWatcher(clientset, cfg)
```

## Pod to RuntimeInstance transform

`PodListerWatcher` produces `meshresource.RuntimeInstanceKind`:

```go
func (p *PodListerWatcher) ResourceKind() coremodel.ResourceKind {
    return meshresource.RuntimeInstanceKind
}
```

Its `TransformFunc` converts Kubernetes Pod objects to `RuntimeInstanceResource`. Key extracted fields include:

- pod name and pod IP
- Dubbo app name from configured annotation or label
- RPC port from configured annotation or label
- mesh/discovery identifier from configured annotation or label
- image, node, workload name and type
- create, start, ready timestamps
- probe ports and pod conditions
- derived phase: starting, running, terminating, crashing, failed, succeeded, unknown

Important output:

```go
res := meshresource.NewRuntimeInstanceResourceWithAttributes(pod.Name, p.getDubboMesh(pod))
res.Spec = &meshproto.RuntimeInstance{
    Name: pod.Name,
    Ip: pod.Status.PodIP,
    RpcPort: rpcPort,
    AppName: appName,
    SourceEngine: p.cfg.ID,
    SourceEngineType: string(p.cfg.Type),
}
```

`PodListerWatcher` also implements `ResourceKeyProvider`. Its `KeyFunc` returns the same key as the transformed RuntimeInstance resource, including tombstone support for delete events.

## Engine informer creation

`initInformers` differs from discovery in two important ways:

1. It uses `resolveInformerKeyFunc(lw)` so a ListWatcher can provide a custom key function.
2. It applies `lw.TransformFunc()` to the informer when present.

```go
informer := controller.NewInformerWithOptions(lw, emitter, rs, resolveInformerKeyFunc(lw), controller.Options{ResyncPeriod: 0})
if lw.TransformFunc() != nil {
    err = informer.SetTransform(lw.TransformFunc())
}
```

This is necessary for Kubernetes Pods because the raw watch object is a Pod, but the stored resource is RuntimeInstance.

## Start and leader election

Memory store starts business logic directly. Non-memory stores may run leader election so only the leader starts informers. On leadership loss, the leader-specific stop channel is closed and informer goroutines exit.

`startBusinessLogic` subscribes engine subscribers once, then starts informers:

```go
if !e.subscribed.Load() {
    for _, sub := range e.subscribers {
        e.subscriptionManager.Subscribe(sub)
    }
    e.subscribed.Store(true)
}
for _, informer := range e.informers {
    go informer.Run(stopCh)
}
```

## RuntimeInstance subscriber

Key file: `pkg/core/engine/subscriber/runtime_instance.go`

The subscriber listens to `meshresource.RuntimeInstanceKind`:

```go
func (s *RuntimeInstanceEventSubscriber) ResourceKind() coremodel.ResourceKind {
    return meshresource.RuntimeInstanceKind
}
```

Event handling:

- Added, updated, replaced, sync: require `newObj`, call `processUpsert`.
- Deleted: require `oldObj`, call `processDelete`.

### Upsert behavior

`processUpsert` first finds a related Instance. For Kubernetes runtime instances, lookup prefers full runtime identity, then name, then IP.

If an Instance exists, runtime fields are merged into it:

```go
meshresource.MergeRuntimeInstanceIntoInstance(rtInstanceRes, instanceResource)
s.instanceStore.Update(instanceResource)
```

If no Instance exists, the subscriber creates a runtime-only Instance only when identity is complete:

```go
if !checkAttributesEnough(rtInstanceRes) {
    return nil
}
instanceRes := meshresource.FromRuntimeInstance(rtInstanceRes)
s.instanceStore.Add(instanceRes)
s.eventEmitter.Send(events.NewResourceChangedEvent(cache.Added, nil, instanceRes))
```

`checkAttributesEnough` requires app name, IP, positive RPC port, non-empty mesh, and mesh not equal to the default mesh.

### Delete behavior

`processDelete` clears runtime source from the related Instance:

- If RPC source remains, update the Instance and emit an Instance updated event.
- If no RPC source remains, delete the Instance and emit an Instance deleted event.

```go
meshresource.ClearRuntimeInstanceFromInstance(instanceResource)
if meshresource.HasRPCInstanceSource(instanceResource) {
    s.instanceStore.Update(instanceResource)
    s.eventEmitter.Send(events.NewResourceChangedEvent(cache.Updated, instanceResource, instanceResource))
    return nil
}
s.instanceStore.Delete(instanceResource)
s.eventEmitter.Send(events.NewResourceChangedEvent(cache.Deleted, instanceResource, nil))
```

## Common failure modes

- Kubernetes config cannot be built from kubeconfig or in-cluster config.
- Pod selector is invalid.
- Pod transform cannot identify app name, RPC port, or mesh.
- RuntimeInstance identity is incomplete, so runtime-only Instance creation is skipped.
- Duplicate Instance matches by name or IP cause subscriber to skip ambiguous merge.
- EventBus dispatch issues should be debugged with `dubbo-admin-events`.

## Review checklist

- Confirm ListWatcher `ResourceKind()` is `RuntimeInstanceKind`.
- Confirm raw Pod keys and transformed RuntimeInstance keys are consistent via `KeyFunc`.
- Confirm transform is set before informer starts.
- Confirm runtime-only Instance creation does not happen for default mesh or incomplete identity.
- Confirm delete keeps RPC-sourced Instances and removes only runtime source.
