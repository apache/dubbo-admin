# EventBus Dispatch Implementation

This reference explains the shared EventBus used by discovery, engine, informers, and subscribers.

## End-to-end call chain

```text
runtime.RegisterComponent(&eventBus{})
  -> Bootstrap initializes eventBus.Init
  -> eventBus.subscriberDir = map[ResourceKind]Subscribers
  -> discovery/engine startBusinessLogic calls Subscribe(subscriber)
  -> informer or subscriber creates ResourceChangedEvent
  -> Emitter.Send(event)
  -> eventBus.Send derives ResourceKind
  -> eventBus finds subscribers for that kind
  -> each subscriber.ProcessEvent(event) runs synchronously
  -> errors are logged and dispatch continues
```

## Interfaces

Key file: `pkg/core/events/eventbus.go`

```go
type Event interface {
    Type() cache.DeltaType
    OldObj() model.Resource
    NewObj() model.Resource
    Context() map[string]string
    String() string
}

type Subscriber interface {
    ResourceKind() model.ResourceKind
    Name() string
    ProcessEvent(event Event) error
}

type Emitter interface {
    Send(event Event)
}

type SubscriptionManager interface {
    Subscribe(subscriber Subscriber) error
    Unsubscribe(subscriber Subscriber) error
}

type EventBus interface {
    Emitter
    SubscriptionManager
}
```

`ResourceChangedEvent` stores delta type, old object, new object, and optional context. Informers usually create it with `events.NewResourceChangedEvent`.

## Component registration

Key file: `pkg/core/events/component.go`

```go
func init() {
    runtime.RegisterComponent(&eventBus{})
}
```

`eventBus` is a runtime component of type `runtime.EventBus`. It has no required dependencies and uses max order for runtime start ordering:

```go
func (b *eventBus) RequiredDependencies() []runtime.ComponentType {
    return []runtime.ComponentType{}
}

func (b *eventBus) Type() runtime.ComponentType {
    return runtime.EventBus
}

func (b *eventBus) Order() int {
    return math.MaxInt
}
```

`Init` creates the subscriber directory:

```go
func (b *eventBus) Init(_ runtime.BuilderContext) error {
    b.subscriberDir = make(map[model.ResourceKind]Subscribers)
    return nil
}
```

`Start` is a no-op.

## Subscribe behavior

Subscribers are grouped by resource kind. Names must be unique within the same resource kind:

```go
func (b *eventBus) Subscribe(subscriber Subscriber) error {
    b.rwMutex.Lock()
    defer b.rwMutex.Unlock()
    subs, exists := b.subscriberDir[subscriber.ResourceKind()]
    if !exists {
        subs = make(Subscribers, 0)
    }
    for _, sub := range subs {
        if sub.Name() == subscriber.Name() {
            return fmt.Errorf("duplicated subscriber name %s, skipped subscribing", subscriber.Name())
        }
    }
    b.subscriberDir[subscriber.ResourceKind()] = append(subs, subscriber)
    return nil
}
```

This means two subscribers may share a name only if they subscribe to different resource kinds. Discovery and engine subscribers typically use prefixes such as `Discovery-` and `Engine-` in `Name()`.

## Unsubscribe behavior

`Unsubscribe` removes a subscriber by resource kind and name:

```go
rk := subscriber.ResourceKind()
name := subscriber.Name()
subs, exists := b.subscriberDir[rk]
...
if sub.Name() == name {
    b.subscriberDir[rk] = append(subs[:i], subs[i+1:]...)
    return nil
}
```

If no subscriber list or no matching name exists, it returns an error.

## Send dispatch algorithm

`Send` uses a read lock and derives the dispatch key from event resources:

```go
var rk model.ResourceKind
if event.NewObj() != nil {
    rk = event.NewObj().ResourceKind()
} else if event.OldObj() != nil {
    rk = event.OldObj().ResourceKind()
}
```

Then it looks up subscribers and calls them synchronously:

```go
subs, exists := b.subscriberDir[rk]
if !exists {
    logger.Infof("no subscriber for resource %s, skipped sending event%v", rk, event)
    return
}
for _, sub := range subs {
    if err := sub.ProcessEvent(event); err != nil {
        logger.Errorf("failed to process event in %s, cause: %s, event: %v", sub.Name(), err.Error(), event)
    }
}
```

Important semantics:

- New object wins over old object for routing.
- Delete events rely on old object because new object is nil.
- Events with both old and new nil cannot be routed and will use the zero resource kind.
- Subscriber errors are logged; they do not stop dispatch to later subscribers.
- Dispatch is synchronous; long subscriber work blocks the sender.

## Common producers

- `pkg/core/controller/informer.go`: emits events after store add, update, or delete.
- `pkg/core/discovery/subscriber/`: emits follow-up Application, Service, or Instance events after deriving resources.
- `pkg/core/engine/subscriber/`: emits follow-up Instance events after RuntimeInstance merge/delete.

## Review checklist

- Confirm delete events provide `OldObj`.
- Confirm subscriber `ResourceKind()` matches the resource being emitted.
- Confirm subscriber `Name()` is unique for that resource kind.
- Do not put business logic in EventBus; keep it in subscribers.
- If changing error behavior, review discovery and engine because both rely on current non-blocking error handling.
