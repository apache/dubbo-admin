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
	"testing"

	"github.com/stretchr/testify/assert"
)

// Mock component for testing
type mockComponent struct {
	typ   ComponentType
	order int
	deps  []ComponentType
}

func (m *mockComponent) Type() ComponentType {
	return m.typ
}
func (m *mockComponent) Order() int {
	return m.order
}
func (m *mockComponent) RequiredDependencies() []ComponentType {
	return m.deps
}
func (m *mockComponent) Init(ctx BuilderContext) error {
	return nil
}
func (m *mockComponent) Start(Runtime, <-chan struct{}) error {
	return nil
}

func TestDependencyGraph_SimpleLinearDependency(t *testing.T) {
	// A -> B -> C
	components := []Component{
		&mockComponent{typ: "A", order: 100, deps: []ComponentType{"B"}},
		&mockComponent{typ: "B", order: 200, deps: []ComponentType{"C"}},
		&mockComponent{typ: "C", order: 300, deps: []ComponentType{}},
	}

	graph := NewDependencyGraph(components)
	sorted, err := graph.TopologicalSort()

	assert.NoError(t, err)
	assert.Equal(t, 3, len(sorted))
	// C should be first, then B, then A
	assert.Equal(t, ComponentType("C"), sorted[0].Type())
	assert.Equal(t, ComponentType("B"), sorted[1].Type())
	assert.Equal(t, ComponentType("A"), sorted[2].Type())
}

func TestDependencyGraph_DiamondDependency(t *testing.T) {
	// A depends on B and C, both B and C depend on D
	//     A
	//    / \
	//   B   C
	//    \ /
	//     D
	components := []Component{
		&mockComponent{typ: "A", order: 100, deps: []ComponentType{"B", "C"}},
		&mockComponent{typ: "B", order: 200, deps: []ComponentType{"D"}},
		&mockComponent{typ: "C", order: 300, deps: []ComponentType{"D"}},
		&mockComponent{typ: "D", order: 400, deps: []ComponentType{}},
	}

	graph := NewDependencyGraph(components)
	sorted, err := graph.TopologicalSort()

	assert.NoError(t, err)
	assert.Equal(t, 4, len(sorted))
	// D must be first
	assert.Equal(t, ComponentType("D"), sorted[0].Type())
	// B and C can be in any order (alphabetically: B before C)
	assert.Equal(t, ComponentType("B"), sorted[1].Type())
	assert.Equal(t, ComponentType("C"), sorted[2].Type())
	// A must be last
	assert.Equal(t, ComponentType("A"), sorted[3].Type())
}

func TestDependencyGraph_CircularDependency_TwoNodes(t *testing.T) {
	// A -> B -> A (circular)
	components := []Component{
		&mockComponent{typ: "A", order: 100, deps: []ComponentType{"B"}},
		&mockComponent{typ: "B", order: 200, deps: []ComponentType{"A"}},
	}

	graph := NewDependencyGraph(components)
	_, err := graph.TopologicalSort()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency detected")
	assert.Contains(t, err.Error(), "A")
	assert.Contains(t, err.Error(), "B")
}

func TestDependencyGraph_CircularDependency_ThreeNodes(t *testing.T) {
	// A -> B -> C -> A (circular)
	components := []Component{
		&mockComponent{typ: "A", order: 100, deps: []ComponentType{"B"}},
		&mockComponent{typ: "B", order: 200, deps: []ComponentType{"C"}},
		&mockComponent{typ: "C", order: 300, deps: []ComponentType{"A"}},
	}

	graph := NewDependencyGraph(components)
	_, err := graph.TopologicalSort()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency detected")
	errMsg := err.Error()
	assert.Contains(t, errMsg, "A")
	assert.Contains(t, errMsg, "B")
	assert.Contains(t, errMsg, "C")
}

func TestDependencyGraph_MissingDependency(t *testing.T) {
	// A depends on non-existent component "X"
	components := []Component{
		&mockComponent{typ: "A", order: 100, deps: []ComponentType{"X"}},
	}

	graph := NewDependencyGraph(components)
	_, err := graph.TopologicalSort()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing dependency")
	assert.Contains(t, err.Error(), "X")
}

func TestDependencyGraph_NoDependencies(t *testing.T) {
	// All components independent, should sort alphabetically (deterministic)
	components := []Component{
		&mockComponent{typ: "A", order: 100, deps: []ComponentType{}},
		&mockComponent{typ: "B", order: 300, deps: []ComponentType{}},
		&mockComponent{typ: "C", order: 200, deps: []ComponentType{}},
	}

	graph := NewDependencyGraph(components)
	sorted, err := graph.TopologicalSort()

	assert.NoError(t, err)
	assert.Equal(t, 3, len(sorted))
	// With no dependencies, components are sorted alphabetically
	assert.Equal(t, ComponentType("A"), sorted[0].Type())
	assert.Equal(t, ComponentType("B"), sorted[1].Type())
	assert.Equal(t, ComponentType("C"), sorted[2].Type())
}

func TestDependencyGraph_ComplexDependencies(t *testing.T) {
	// Simulating real dubbo-admin components
	// EventBus (no deps)
	// Store -> EventBus
	// Discovery -> EventBus, Store
	// Engine -> EventBus, Store
	// Manager -> Store
	// Console -> Manager
	components := []Component{
		&mockComponent{typ: "EventBus", order: 1000, deps: []ComponentType{}},
		&mockComponent{typ: "Store", order: 900, deps: []ComponentType{"EventBus"}},
		&mockComponent{typ: "Discovery", order: 800, deps: []ComponentType{"EventBus", "Store"}},
		&mockComponent{typ: "Engine", order: 700, deps: []ComponentType{"EventBus", "Store"}},
		&mockComponent{typ: "Manager", order: 600, deps: []ComponentType{"Store"}},
		&mockComponent{typ: "Console", order: 500, deps: []ComponentType{"Manager"}},
	}

	graph := NewDependencyGraph(components)
	sorted, err := graph.TopologicalSort()

	assert.NoError(t, err)
	assert.Equal(t, 6, len(sorted))

	// Create a map for easier verification
	orderMap := make(map[ComponentType]int)
	for i, comp := range sorted {
		orderMap[comp.Type()] = i
	}

	// Verify dependency constraints
	assert.Less(t, orderMap["EventBus"], orderMap["Store"], "EventBus must initialize before Store")
	assert.Less(t, orderMap["EventBus"], orderMap["Discovery"], "EventBus must initialize before Discovery")
	assert.Less(t, orderMap["Store"], orderMap["Discovery"], "Store must initialize before Discovery")
	assert.Less(t, orderMap["EventBus"], orderMap["Engine"], "EventBus must initialize before Engine")
	assert.Less(t, orderMap["Store"], orderMap["Engine"], "Store must initialize before Engine")
	assert.Less(t, orderMap["Store"], orderMap["Manager"], "Store must initialize before Manager")
	assert.Less(t, orderMap["Manager"], orderMap["Console"], "Manager must initialize before Console")
}

func TestDependencyGraph_SelfDependency(t *testing.T) {
	// Component depends on itself (edge case)
	components := []Component{
		&mockComponent{typ: "A", order: 100, deps: []ComponentType{"A"}},
	}

	graph := NewDependencyGraph(components)
	_, err := graph.TopologicalSort()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency detected")
	assert.Contains(t, err.Error(), "A -> A")
}

func TestDependencyGraph_MultipleDependenciesSameLevel(t *testing.T) {
	// A depends on B, C, D (all at same level)
	// B, C, D have no dependencies
	components := []Component{
		&mockComponent{typ: "A", order: 100, deps: []ComponentType{"B", "C", "D"}},
		&mockComponent{typ: "B", order: 200, deps: []ComponentType{}},
		&mockComponent{typ: "C", order: 300, deps: []ComponentType{}},
		&mockComponent{typ: "D", order: 400, deps: []ComponentType{}},
	}

	graph := NewDependencyGraph(components)
	sorted, err := graph.TopologicalSort()

	assert.NoError(t, err)
	assert.Equal(t, 4, len(sorted))

	// Create a map for easier verification
	orderMap := make(map[ComponentType]int)
	for i, comp := range sorted {
		orderMap[comp.Type()] = i
	}

	// All dependencies must come before A
	assert.Less(t, orderMap["B"], orderMap["A"])
	assert.Less(t, orderMap["C"], orderMap["A"])
	assert.Less(t, orderMap["D"], orderMap["A"])

	// A must be last
	assert.Equal(t, ComponentType("A"), sorted[3].Type())
}

func TestDependencyGraph_EmptyComponents(t *testing.T) {
	// Empty component list
	components := []Component{}

	graph := NewDependencyGraph(components)
	sorted, err := graph.TopologicalSort()

	assert.NoError(t, err)
	assert.Equal(t, 0, len(sorted))
}

func TestDependencyGraph_SingleComponent(t *testing.T) {
	// Single component with no dependencies
	components := []Component{
		&mockComponent{typ: "A", order: 100, deps: []ComponentType{}},
	}

	graph := NewDependencyGraph(components)
	sorted, err := graph.TopologicalSort()

	assert.NoError(t, err)
	assert.Equal(t, 1, len(sorted))
	assert.Equal(t, ComponentType("A"), sorted[0].Type())
}
