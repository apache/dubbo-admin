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

package bootstrap

import (
	"testing"

	"github.com/apache/dubbo-admin/pkg/core/runtime"
)

// Mock component for testing
type mockComponent struct {
	name         runtime.ComponentType
	dependencies []runtime.ComponentType
	initialized  bool
}

func (m *mockComponent) Type() runtime.ComponentType {
	return m.name
}

func (m *mockComponent) Order() int {
	return 0
}

func (m *mockComponent) Dependencies() []runtime.ComponentType {
	return m.dependencies
}

func (m *mockComponent) Init(ctx runtime.BuilderContext) error {
	m.initialized = true
	return nil
}

func (m *mockComponent) Start(rt runtime.Runtime, stop <-chan struct{}) error {
	return nil
}

func TestBuildComponentDAG_NoDependencies(t *testing.T) {
	components := []runtime.Component{
		&mockComponent{name: "comp1", dependencies: nil},
		&mockComponent{name: "comp2", dependencies: nil},
		&mockComponent{name: "comp3", dependencies: nil},
	}

	dag, err := buildComponentDAG(components)
	if err != nil {
		t.Fatalf("buildComponentDAG failed: %v", err)
	}

	if dag == nil {
		t.Fatal("DAG should not be nil")
	}

	// All components should be roots since they have no dependencies
	roots := dag.GetRoots()
	if len(roots) != 3 {
		t.Errorf("Expected 3 root components, got %d", len(roots))
	}
}

func TestBuildComponentDAG_WithDependencies(t *testing.T) {
	components := []runtime.Component{
		&mockComponent{name: "comp1", dependencies: nil},
		&mockComponent{name: "comp2", dependencies: []runtime.ComponentType{"comp1"}},
		&mockComponent{name: "comp3", dependencies: []runtime.ComponentType{"comp1", "comp2"}},
	}

	dag, err := buildComponentDAG(components)
	if err != nil {
		t.Fatalf("buildComponentDAG failed: %v", err)
	}

	// Only comp1 should be a root
	roots := dag.GetRoots()
	if len(roots) != 1 {
		t.Errorf("Expected 1 root component, got %d", len(roots))
	}

	if _, exists := roots["comp1"]; !exists {
		t.Error("comp1 should be a root component")
	}

	// Verify edges
	children, err := dag.GetChildren("comp1")
	if err != nil {
		t.Fatalf("GetChildren failed: %v", err)
	}
	if len(children) != 2 {
		t.Errorf("comp1 should have 2 children, got %d", len(children))
	}
}

func TestBuildComponentDAG_CircularDependency(t *testing.T) {
	// Create components with circular dependency
	components := []runtime.Component{
		&mockComponent{name: "comp1", dependencies: []runtime.ComponentType{"comp2"}},
		&mockComponent{name: "comp2", dependencies: []runtime.ComponentType{"comp1"}},
	}

	_, err := buildComponentDAG(components)
	if err == nil {
		t.Error("buildComponentDAG should fail with circular dependency")
	}
}

func TestGetInitializationOrder_LinearDependency(t *testing.T) {
	// comp3 -> comp2 -> comp1 (comp3 depends on comp2, comp2 depends on comp1)
	components := []runtime.Component{
		&mockComponent{name: "comp1", dependencies: nil},
		&mockComponent{name: "comp2", dependencies: []runtime.ComponentType{"comp1"}},
		&mockComponent{name: "comp3", dependencies: []runtime.ComponentType{"comp2"}},
	}

	dag, err := buildComponentDAG(components)
	if err != nil {
		t.Fatalf("buildComponentDAG failed: %v", err)
	}

	order, err := getInitializationOrder(dag)
	if err != nil {
		t.Fatalf("getInitializationOrder failed: %v", err)
	}

	if len(order) != 3 {
		t.Fatalf("Expected 3 components in order, got %d", len(order))
	}

	// Verify order: comp1 should come before comp2, comp2 before comp3
	orderMap := make(map[runtime.ComponentType]int)
	for i, comp := range order {
		orderMap[comp.Type()] = i
	}

	if orderMap["comp1"] >= orderMap["comp2"] {
		t.Error("comp1 should be initialized before comp2")
	}
	if orderMap["comp2"] >= orderMap["comp3"] {
		t.Error("comp2 should be initialized before comp3")
	}
}

func TestGetInitializationOrder_DiamondDependency(t *testing.T) {
	// Diamond dependency:
	//     comp1
	//    /     \
	// comp2   comp3
	//    \     /
	//     comp4
	components := []runtime.Component{
		&mockComponent{name: "comp1", dependencies: nil},
		&mockComponent{name: "comp2", dependencies: []runtime.ComponentType{"comp1"}},
		&mockComponent{name: "comp3", dependencies: []runtime.ComponentType{"comp1"}},
		&mockComponent{name: "comp4", dependencies: []runtime.ComponentType{"comp2", "comp3"}},
	}

	dag, err := buildComponentDAG(components)
	if err != nil {
		t.Fatalf("buildComponentDAG failed: %v", err)
	}

	order, err := getInitializationOrder(dag)
	if err != nil {
		t.Fatalf("getInitializationOrder failed: %v", err)
	}

	if len(order) != 4 {
		t.Fatalf("Expected 4 components in order, got %d", len(order))
	}

	// Verify order constraints
	orderMap := make(map[runtime.ComponentType]int)
	for i, comp := range order {
		orderMap[comp.Type()] = i
	}

	// comp1 must be first
	if orderMap["comp1"] != 0 {
		t.Error("comp1 should be initialized first")
	}

	// comp2 and comp3 must come after comp1
	if orderMap["comp2"] <= orderMap["comp1"] {
		t.Error("comp2 should be initialized after comp1")
	}
	if orderMap["comp3"] <= orderMap["comp1"] {
		t.Error("comp3 should be initialized after comp1")
	}

	// comp4 must come after both comp2 and comp3
	if orderMap["comp4"] <= orderMap["comp2"] {
		t.Error("comp4 should be initialized after comp2")
	}
	if orderMap["comp4"] <= orderMap["comp3"] {
		t.Error("comp4 should be initialized after comp3")
	}
}

func TestGetInitializationOrder_MultipleRoots(t *testing.T) {
	// Multiple independent trees
	components := []runtime.Component{
		&mockComponent{name: "comp1", dependencies: nil},
		&mockComponent{name: "comp2", dependencies: []runtime.ComponentType{"comp1"}},
		&mockComponent{name: "comp3", dependencies: nil},
		&mockComponent{name: "comp4", dependencies: []runtime.ComponentType{"comp3"}},
	}

	dag, err := buildComponentDAG(components)
	if err != nil {
		t.Fatalf("buildComponentDAG failed: %v", err)
	}

	order, err := getInitializationOrder(dag)
	if err != nil {
		t.Fatalf("getInitializationOrder failed: %v", err)
	}

	if len(order) != 4 {
		t.Fatalf("Expected 4 components in order, got %d", len(order))
	}

	// Verify order constraints
	orderMap := make(map[runtime.ComponentType]int)
	for i, comp := range order {
		orderMap[comp.Type()] = i
	}

	// comp1 must come before comp2
	if orderMap["comp1"] >= orderMap["comp2"] {
		t.Error("comp1 should be initialized before comp2")
	}

	// comp3 must come before comp4
	if orderMap["comp3"] >= orderMap["comp4"] {
		t.Error("comp3 should be initialized before comp4")
	}
}

func TestGetInitializationOrder_MissingDependency(t *testing.T) {
	// comp2 depends on comp1, but comp1 is not in the list
	components := []runtime.Component{
		&mockComponent{name: "comp2", dependencies: []runtime.ComponentType{"comp1"}},
	}

	_, err := buildComponentDAG(components)
	if err == nil {
		t.Error("buildComponentDAG should fail when dependency is missing")
	}
}

func TestGetInitializationOrder_RealWorldScenario(t *testing.T) {
	// Simulate the real components in dubbo-admin
	components := []runtime.Component{
		&mockComponent{name: runtime.EventBus, dependencies: nil},
		&mockComponent{name: runtime.ResourceStore, dependencies: nil},
		&mockComponent{name: runtime.Console, dependencies: nil},
		&mockComponent{name: runtime.ResourceDiscovery, dependencies: []runtime.ComponentType{runtime.EventBus, runtime.ResourceStore}},
		&mockComponent{name: runtime.ResourceEngine, dependencies: []runtime.ComponentType{runtime.EventBus, runtime.ResourceStore}},
		&mockComponent{name: runtime.ResourceManager, dependencies: []runtime.ComponentType{runtime.ResourceStore}},
		&mockComponent{name: "counter manager", dependencies: []runtime.ComponentType{runtime.EventBus}},
	}

	dag, err := buildComponentDAG(components)
	if err != nil {
		t.Fatalf("buildComponentDAG failed: %v", err)
	}

	order, err := getInitializationOrder(dag)
	if err != nil {
		t.Fatalf("getInitializationOrder failed: %v", err)
	}

	if len(order) != 7 {
		t.Fatalf("Expected 7 components in order, got %d", len(order))
	}

	// Build order map for validation
	orderMap := make(map[runtime.ComponentType]int)
	for i, comp := range order {
		orderMap[comp.Type()] = i
		t.Logf("%d: %s", i, comp.Type())
	}

	// Validate dependency constraints
	// EventBus must be before ResourceDiscovery, ResourceEngine, and counter manager
	if orderMap[runtime.EventBus] >= orderMap[runtime.ResourceDiscovery] {
		t.Error("EventBus should be initialized before ResourceDiscovery")
	}
	if orderMap[runtime.EventBus] >= orderMap[runtime.ResourceEngine] {
		t.Error("EventBus should be initialized before ResourceEngine")
	}
	if orderMap[runtime.EventBus] >= orderMap["counter manager"] {
		t.Error("EventBus should be initialized before counter manager")
	}

	// ResourceStore must be before ResourceDiscovery, ResourceEngine, and ResourceManager
	if orderMap[runtime.ResourceStore] >= orderMap[runtime.ResourceDiscovery] {
		t.Error("ResourceStore should be initialized before ResourceDiscovery")
	}
	if orderMap[runtime.ResourceStore] >= orderMap[runtime.ResourceEngine] {
		t.Error("ResourceStore should be initialized before ResourceEngine")
	}
	if orderMap[runtime.ResourceStore] >= orderMap[runtime.ResourceManager] {
		t.Error("ResourceStore should be initialized before ResourceManager")
	}
}
