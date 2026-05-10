# Store Indexing and Query Implementation

This reference explains how dubbo-admin routes resources to stores, registers indexes, and executes indexed queries in the memory store.

## End-to-end call chain

```text
runtime.RegisterComponent(newStoreComponent)
  -> Bootstrap initializes storeComponent.Init
  -> Init gets store config
  -> FactoryRegistry().GetStoreFactory(cfg.Type)
  -> ResourceSchemaRegistry().AllResourceKinds()
  -> factory.New(kind, cfg)
  -> store.Init(ctx)
  -> storeComponent.ResourceKindRoute(kind)
  -> caller invokes ResourceStore Add/Update/Delete/ListByIndexes/PageListByIndexes
  -> memory resourceStore delegates exact indexes to cache.Indexer
  -> memory resourceStore delegates prefix indexes to radix trees
```

## Store component routing

Key file: `pkg/core/store/component.go`

The store component is registered as `runtime.ResourceStore` and depends on EventBus:

```go
func init() {
    runtime.RegisterComponent(newStoreComponent())
}

func (sc *storeComponent) RequiredDependencies() []runtime.ComponentType {
    return []runtime.ComponentType{runtime.EventBus}
}
```

During `Init`, it creates one managed store per registered resource kind:

```go
cfg := ctx.Config().Store
factory, err := FactoryRegistry().GetStoreFactory(cfg.Type)
for _, kind := range coremodel.ResourceSchemaRegistry().AllResourceKinds() {
    store, err := factory.New(kind, cfg)
    sc.stores[kind] = store
    err = store.Init(ctx)
}
```

Routing is by resource kind:

```go
func (sc *storeComponent) ResourceKindRoute(k coremodel.ResourceKind) (ResourceStore, error) {
    if store, exists := sc.stores[k]; exists {
        return store, nil
    }
    return nil, fmt.Errorf("%s is not supported by store yet", k)
}
```

## ResourceStore contract

Key file: `pkg/core/store/store.go`

```go
type ResourceStore interface {
    Indexer
    GetByKeys(keys []string) ([]model.Resource, error)
    ListByIndexes(indexes []index.IndexCondition) ([]model.Resource, error)
    PageListByIndexes(indexes []index.IndexCondition, pq model.PageReq) (*model.PageData[model.Resource], error)
}
```

`ManagedResourceStore` adds runtime lifecycle to `ResourceStore`, so each concrete store participates in bootstrap and start.

## Index registration

Key directory: `pkg/core/store/index/`

Indexers register during package initialization with `RegisterIndexers`:

```go
func RegisterIndexers(rk model.ResourceKind, indexers cache.Indexers) {
    indexRegistry.Register(rk, indexers)
}
```

`ByMeshIndex` is registered for mesh resources in `common.go`. Resource-specific files register indexes such as service name, provider service key, consumer service key, instance IP, and runtime instance IP.

When adding a query, verify that the target resource kind has a registered index for every `IndexCondition` used.

## Memory store initialization

Key file: `pkg/store/memory/store.go`

Memory store wraps a Kubernetes `cache.Indexer` and maintains radix trees for prefix lookups:

```go
type resourceStore struct {
    rk          coremodel.ResourceKind
    storeProxy  cache.Indexer
    prefixTrees map[string]*radix.Tree
    treesMu     sync.RWMutex
}
```

`Init` loads registered indexers for the resource kind and initializes one radix tree per index:

```go
indexers := index.IndexersRegistry().Indexers(rs.rk)
rs.storeProxy = cache.NewIndexer(keyFunc, indexers)
rs.prefixTrees = make(map[string]*radix.Tree)
for indexName := range indexers {
    rs.prefixTrees[indexName] = radix.New()
}
```

## Write path and prefix tree maintenance

`Add`, `Update`, `Delete`, and `Replace` keep radix trees in sync with the underlying indexer.

Add:

```go
rs.storeProxy.Add(obj)
rs.addToTrees(resource)
```

Update removes old index values and adds new ones only after successful store update:

```go
oldObj, exists, err := rs.storeProxy.Get(r)
rs.storeProxy.Update(obj)
if oldRes != nil { rs.removeFromTrees(oldRes) }
rs.addToTrees(r)
```

Delete:

```go
rs.storeProxy.Delete(obj)
rs.removeFromTrees(resource)
```

Dynamic indexes are supported through `AddIndexers`; it adds missing radix trees for new indexers.

## Query path

`ListByIndexes` and `PageListByIndexes` both start with `getKeysByIndexes`.

```go
func (rs *resourceStore) getKeysByIndexes(indexes []index.IndexCondition) ([]string, error) {
    if len(indexes) == 0 {
        return []string{}, nil
    }
    keySet := set.New[string]()
    first := true
    for _, condition := range indexes {
        switch condition.Operator {
        case index.Equals:
            keys, err = rs.storeProxy.IndexKeys(condition.IndexName, condition.Value)
        case index.HasPrefix:
            keys, err = rs.getKeysByPrefix(condition.IndexName, condition.Value)
        default:
            return nil, bizerror.New(bizerror.InvalidArgument, "operator not yet supported: "+string(condition.Operator))
        }
        if first { keySet = set.FromSlice(keys) } else { keySet = keySet.Intersection(set.FromSlice(keys)) }
    }
    return keySet.ToSlice(), nil
}
```

Important semantics:

- Empty conditions return no keys, not all resources.
- Multiple conditions are AND semantics through intersection.
- Unsupported operators return invalid argument errors.

`ListByIndexes` loads resources by keys and sorts resources by `ResourceKey()`. `PageListByIndexes` sorts keys, applies offset and page size, loads resources, and returns `coremodel.PageData`.

## Prefix lookup implementation

Radix keys are stored as:

```text
indexValue/resourceKey
```

`resourceKey` itself contains `/` as `mesh/name`, so prefix extraction uses the first slash:

```go
idx := strings.Index(k, "/")
keys = append(keys, k[idx+1:])
```

Review prefix queries carefully because a wrong separator assumption can corrupt keys.

## DB-backed store and leader election support

`storeComponent` implements `leader.DBSource` by reflectively finding a store pool with `GetDB`. Discovery and engine use this to decide whether leader election is available for non-memory stores.

## Common failure modes

- Query uses an index name that was never registered for the resource kind.
- Mesh-scoped query forgets `ByMeshIndex` and returns cross-mesh data.
- Prefix tree is not updated after update/delete/dynamic index addition.
- Empty index conditions are expected to return no results.
- Multiple conditions unexpectedly narrow results because they are intersections.

## Review checklist

- Confirm every `IndexCondition` has a registered index for the resource kind.
- Confirm mesh scoping is explicit where required.
- Confirm write paths maintain both `storeProxy` and `prefixTrees`.
- Confirm pagination uses stable key ordering.
- Confirm caller expectations match empty-condition behavior.
