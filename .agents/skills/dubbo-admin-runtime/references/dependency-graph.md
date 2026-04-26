# Runtime Dependency Graph

This reference explains how dubbo-admin sorts runtime components by `RequiredDependencies()`.

## Key file

- `pkg/core/runtime/dependency_graph.go`

## End-to-end algorithm

```text
NewDependencyGraph(components)
  -> map each ComponentType to Component
  -> initialize adjacency list for all component types
  -> for each component dependency, add edge dependency -> component
TopologicalSort()
  -> validate all dependencies exist
  -> compute indegree = len(RequiredDependencies())
  -> enqueue components with indegree 0
  -> sort queue alphabetically for deterministic ordering
  -> pop current, append component to result
  -> decrement indegree for dependents
  -> detect cycle if result length is incomplete
```

## Graph shape

`DependencyGraph` stores reverse adjacency:

```go
type DependencyGraph struct {
    components map[ComponentType]Component
    adjList    map[ComponentType][]ComponentType
}
```

When A depends on B, the stored edge is B to A:

```go
for _, comp := range components {
    for _, dependency := range comp.RequiredDependencies() {
        dg.adjList[dependency] = append(dg.adjList[dependency], comp.Type())
    }
}
```

This makes topological output match initialization order: dependencies appear before dependents.

## Validation step

Before sorting, every declared dependency must exist in the component set:

```go
func (dg *DependencyGraph) validate() error {
    for _, comp := range dg.components {
        for _, dependency := range comp.RequiredDependencies() {
            if _, exists := dg.components[dependency]; !exists {
                return bizerror.Wrap(
                    fmt.Errorf("missing dependency %q", dependency),
                    bizerror.ConfigError,
                    fmt.Sprintf("component %q requires missing dependency %q", comp.Type(), dependency),
                )
            }
        }
    }
    return nil
}
```

A missing dependency is a configuration/registration problem, not a sorting problem.

## Sorting step

In-degree is dependency count:

```go
indegree[comp.Type()] = len(comp.RequiredDependencies())
```

The queue is sorted every iteration:

```go
sort.Strings(queue)
current := queue[0]
queue = queue[1:]
```

This gives deterministic order among otherwise independent components.

## Cycle detection

If the sorted result does not include every component, `findCycle` uses DFS through `RequiredDependencies()` to extract a cycle. `newCircularDependencyError` formats it as:

```text
A -> B -> C -> A
```

and returns a `bizerror.ConfigError`.

## Review checks

- Add dependencies to `RequiredDependencies()`, not to comments or `Order()`.
- Keep the alphabetic deterministic queue behavior unless there is a strong reason to change reproducibility.
- Missing dependency errors should name both dependent and dependency.
- Cycle errors should remain actionable and point to `RequiredDependencies()`.
- If two components are independent, do not rely on their relative initialization order.
