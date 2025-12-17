/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package runtime

import (
	"fmt"
	"math"
	"strings"
)

// DependencyGraph represents the dependency relationships between components
// The adjacency list is stored in reverse: if A depends on B, we store B -> A
// This makes topological sort return components in initialization order
type DependencyGraph struct {
	components map[ComponentType]Component
	adjList    map[ComponentType][]ComponentType // reverse adjacency list: dependent <- dependee
}

// NewDependencyGraph creates a new dependency graph from components
func NewDependencyGraph(components []Component) *DependencyGraph {
	dg := &DependencyGraph{
		components: make(map[ComponentType]Component),
		adjList:    make(map[ComponentType][]ComponentType),
	}

	// Initialize adjacency list for all components
	for _, comp := range components {
		dg.components[comp.Type()] = comp
		dg.adjList[comp.Type()] = []ComponentType{}
	}

	// Build reverse adjacency list
	// If component A depends on B, we create edge B -> A
	// This way, the adjacency list represents "who depends on me"
	for _, comp := range components {
		if dep, ok := comp.(ComponentWithDependencies); ok {
			for _, dependency := range dep.RequiredDependencies() {
				// dependency -> comp.Type()
				// Meaning: dependency is required by comp.Type()
				dg.adjList[dependency] = append(dg.adjList[dependency], comp.Type())
			}
		}
	}

	return dg
}

// TopologicalSort performs topological sort using Kahn's algorithm
// Returns components in initialization order (dependencies first)
func (dg *DependencyGraph) TopologicalSort() ([]Component, error) {
	// 1. Validate all dependencies exist
	if err := dg.validate(); err != nil {
		return nil, err
	}

	// 2. Calculate in-degree for each node
	// In-degree = number of dependencies
	indegree := make(map[ComponentType]int)
	for typ := range dg.components {
		indegree[typ] = 0
	}

	// Count in-degrees based on original dependencies
	for _, comp := range dg.components {
		if dep, ok := comp.(ComponentWithDependencies); ok {
			indegree[comp.Type()] = len(dep.RequiredDependencies())
		}
	}

	// 3. Find all nodes with in-degree 0 (no dependencies)
	queue := []ComponentType{}
	for typ, deg := range indegree {
		if deg == 0 {
			queue = append(queue, typ)
		}
	}

	// 4. Process nodes in topological order
	result := []Component{}
	visited := make(map[ComponentType]bool)

	for len(queue) > 0 {
		// Pop node with highest Order() value (for deterministic ordering)
		current := dg.popWithHighestOrder(queue)
		queue = dg.removeFromQueue(queue, current)

		result = append(result, dg.components[current])
		visited[current] = true

		// Process all components that depend on current
		for _, dependent := range dg.adjList[current] {
			indegree[dependent]--
			if indegree[dependent] == 0 && !visited[dependent] {
				queue = append(queue, dependent)
			}
		}
	}

	// 5. Check for circular dependencies
	if len(result) != len(dg.components) {
		cycle := dg.findCycle()
		return nil, &CircularDependencyError{Cycle: cycle}
	}

	return result, nil
}

// validate checks that all declared dependencies exist
func (dg *DependencyGraph) validate() error {
	for _, comp := range dg.components {
		if dep, ok := comp.(ComponentWithDependencies); ok {
			for _, dependency := range dep.RequiredDependencies() {
				if _, exists := dg.components[dependency]; !exists {
					return fmt.Errorf(
						"component %q requires missing dependency %q\n"+
							"Hint: Make sure %q is registered in ComponentRegistry()",
						comp.Type(), dependency, dependency,
					)
				}
			}
		}
	}
	return nil
}

// popWithHighestOrder returns the component with highest Order() value from queue
// This ensures deterministic ordering when multiple components have the same dependency level
func (dg *DependencyGraph) popWithHighestOrder(queue []ComponentType) ComponentType {
	if len(queue) == 1 {
		return queue[0]
	}

	maxOrder := math.MinInt
	maxType := queue[0]

	for _, typ := range queue {
		order := dg.components[typ].Order()
		if order > maxOrder {
			maxOrder = order
			maxType = typ
		}
	}

	return maxType
}

// removeFromQueue removes the first occurrence of element from queue
func (dg *DependencyGraph) removeFromQueue(queue []ComponentType, element ComponentType) []ComponentType {
	result := make([]ComponentType, 0, len(queue)-1)
	found := false
	for _, v := range queue {
		if v == element && !found {
			found = true
			continue
		}
		result = append(result, v)
	}
	return result
}

// findCycle uses DFS to find a circular dependency
func (dg *DependencyGraph) findCycle() []ComponentType {
	visited := make(map[ComponentType]bool)
	recStack := make(map[ComponentType]bool)
	var cycle []ComponentType

	var dfs func(ComponentType) bool
	dfs = func(node ComponentType) bool {
		visited[node] = true
		recStack[node] = true
		cycle = append(cycle, node)

		// Visit dependencies (not dependents)
		if dep, ok := dg.components[node].(ComponentWithDependencies); ok {
			for _, dependency := range dep.RequiredDependencies() {
				if !visited[dependency] {
					if dfs(dependency) {
						return true
					}
				} else if recStack[dependency] {
					// Found cycle, extract the cycle path
					for i, typ := range cycle {
						if typ == dependency {
							cycle = cycle[i:]
							return true
						}
					}
				}
			}
		}

		recStack[node] = false
		cycle = cycle[:len(cycle)-1]
		return false
	}

	for typ := range dg.components {
		if !visited[typ] {
			if dfs(typ) {
				return cycle
			}
		}
	}

	return nil
}

// CircularDependencyError represents a circular dependency error
type CircularDependencyError struct {
	Cycle []ComponentType
}

func (e *CircularDependencyError) Error() string {
	return fmt.Sprintf(
		"circular dependency detected: %s\n"+
			"Please check the RequiredDependencies() methods of these components and break the cycle",
		e.CyclePath(),
	)
}

// CyclePath returns a human-readable representation of the cycle
func (e *CircularDependencyError) CyclePath() string {
	if len(e.Cycle) == 0 {
		return ""
	}
	// Close the cycle
	path := make([]string, len(e.Cycle)+1)
	for i, typ := range e.Cycle {
		path[i] = string(typ)
	}
	path[len(e.Cycle)] = string(e.Cycle[0])
	return strings.Join(path, " -> ")
}
