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
	"sort"
	"strings"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
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
		for _, dependency := range comp.RequiredDependencies() {
			// dependency -> comp.Type()
			// Meaning: dependency is required by comp.Type()
			dg.adjList[dependency] = append(dg.adjList[dependency], comp.Type())
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
	for _, comp := range dg.components {
		indegree[comp.Type()] = len(comp.RequiredDependencies())
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
		// Sort queue alphabetically for deterministic ordering
		sort.Strings(queue)

		// Pop the first element
		current := queue[0]
		queue = queue[1:]

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
		return nil, newCircularDependencyError(cycle)
	}

	return result, nil
}

// validate checks that all declared dependencies exist
func (dg *DependencyGraph) validate() error {
	for _, comp := range dg.components {
		for _, dependency := range comp.RequiredDependencies() {
			if _, exists := dg.components[dependency]; !exists {
				return bizerror.Wrap(
					fmt.Errorf("missing dependency %q", dependency),
					bizerror.ConfigError,
					fmt.Sprintf(
						"component %q requires missing dependency %q. "+
							"Hint: Make sure %q is registered in ComponentRegistry()",
						comp.Type(), dependency, dependency,
					),
				)
			}
		}
	}
	return nil
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
		for _, dependency := range dg.components[node].RequiredDependencies() {
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

// newCircularDependencyError creates a circular dependency error
func newCircularDependencyError(cycle []ComponentType) error {
	if len(cycle) == 0 {
		return bizerror.New(bizerror.ConfigError, "circular dependency detected")
	}

	// Format cycle path: A -> B -> C -> A
	path := make([]string, len(cycle)+1)
	for i, typ := range cycle {
		path[i] = string(typ)
	}
	path[len(cycle)] = string(cycle[0])
	cyclePath := strings.Join(path, " -> ")

	return bizerror.New(
		bizerror.ConfigError,
		fmt.Sprintf(
			"circular dependency detected: %s. "+
				"Please check the RequiredDependencies() methods of these components and break the cycle",
			cyclePath,
		),
	)
}
