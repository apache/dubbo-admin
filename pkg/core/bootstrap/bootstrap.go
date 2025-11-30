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
	"context"
	"fmt"

	"github.com/heimdalr/dag"
	"github.com/pkg/errors"

	"github.com/apache/dubbo-admin/pkg/config/app"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
)

func Bootstrap(appCtx context.Context, cfg app.AdminConfig) (runtime.Runtime, error) {
	builder, err := runtime.BuilderFor(appCtx, cfg)
	if err != nil {
		return nil, err
	}

	// Get all registered components
	allComponents := runtime.ComponentRegistry().AllComponents()

	// Build dependency DAG
	componentDAG, err := buildComponentDAG(allComponents)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build component dependency graph")
	}

	// Get initialization order using topological sort
	initOrder, err := getInitializationOrder(componentDAG)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve component dependencies")
	}

	// Initialize components in dependency order
	for _, comp := range initOrder {
		if err := initAndActivateComponent(builder, comp); err != nil {
			return nil, err
		}
	}

	rt, err := builder.Build()
	if err != nil {
		return nil, err
	}
	return rt, nil
}

// buildComponentDAG builds a directed acyclic graph of component dependencies
func buildComponentDAG(components []runtime.Component) (*dag.DAG, error) {
	d := dag.NewDAG()

	// Add all components as vertices with ComponentType as ID
	for _, comp := range components {
		if err := d.AddVertexByID(comp.Type(), comp); err != nil {
			return nil, errors.Wrapf(err, "failed to add component %s to DAG", comp.Type())
		}
	}

	// Add edges based on dependencies
	for _, comp := range components {
		deps := comp.Dependencies()
		if deps == nil {
			continue
		}
		for _, dep := range deps {
			// Add edge from dependency to dependent
			// If A depends on B, we add edge B -> A (B must be initialized before A)
			if err := d.AddEdge(dep, comp.Type()); err != nil {
				return nil, errors.Wrapf(err, "failed to add dependency edge from %s to %s", dep, comp.Type())
			}
		}
	}

	return d, nil
}

// getInitializationOrder returns components in the order they should be initialized
// using Kahn's algorithm for topological sort
func getInitializationOrder(d *dag.DAG) ([]runtime.Component, error) {
	allVertices := d.GetVertices()

	// Calculate in-degree for each vertex
	inDegree := make(map[string]int)
	for vertexID := range allVertices {
		parents, err := d.GetParents(vertexID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get parents of %s", vertexID)
		}
		inDegree[vertexID] = len(parents)
	}

	// Queue for vertices with in-degree 0
	queue := make([]string, 0)
	for vertexID := range allVertices {
		if inDegree[vertexID] == 0 {
			queue = append(queue, vertexID)
		}
	}

	if len(queue) == 0 {
		return nil, fmt.Errorf("no root components found, possible circular dependency")
	}

	// Process vertices in topological order
	var initOrder []runtime.Component

	for len(queue) > 0 {
		// Dequeue
		currentID := queue[0]
		queue = queue[1:]

		// Get the component
		vertex, err := d.GetVertex(currentID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get vertex %s", currentID)
		}

		comp, ok := vertex.(runtime.Component)
		if !ok {
			return nil, fmt.Errorf("vertex %s is not a Component", currentID)
		}

		initOrder = append(initOrder, comp)

		// Get children and reduce their in-degree
		children, err := d.GetChildren(currentID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get children of %s", currentID)
		}

		for childID := range children {
			inDegree[childID]--
			if inDegree[childID] == 0 {
				queue = append(queue, childID)
			}
		}
	}

	// Check if all vertices have been processed (detect cycles)
	if len(initOrder) != len(allVertices) {
		return nil, fmt.Errorf("circular dependency detected: processed %d components, expected %d",
			len(initOrder), len(allVertices))
	}

	return initOrder, nil
}

func initAndActivateComponent(builder *runtime.Builder, comp runtime.Component) error {
	logger.Infof("initializing %s ...", comp.Type())
	if err := comp.Init(builder); err != nil {
		return err
	}
	logger.Infof("%s initialized successfully", comp.Type())
	if err := builder.ActivateComponent(comp); err != nil {
		return errors.Wrapf(err, "failed to activate %s", comp.Type())
	}
	return nil
}
