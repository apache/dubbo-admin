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

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/config/app"
	"github.com/apache/dubbo-admin/pkg/console/counter"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/diagnostics"
)

func Bootstrap(appCtx context.Context, cfg app.AdminConfig) (runtime.Runtime, error) {
	builder, err := runtime.BuilderFor(appCtx, cfg)
	if err != nil {
		return nil, err
	}

	// Use smart bootstrapper for intelligent component initialization
	bootstrapper := NewSmartBootstrapper(builder)

	// Initialize all components in dependency order
	if err := bootstrapper.bootstrapComponents(appCtx, cfg); err != nil {
		return nil, err
	}

	// Build and return runtime
	rt, err := builder.Build()
	if err != nil {
		return nil, err
	}
	return rt, nil
}

// SmartBootstrapper handles intelligent component initialization
type SmartBootstrapper struct {
	builder *runtime.Builder
}

// NewSmartBootstrapper creates a new smart bootstrapper
func NewSmartBootstrapper(builder *runtime.Builder) *SmartBootstrapper {
	return &SmartBootstrapper{
		builder: builder,
	}
}

// bootstrapComponents initializes all components in dependency order
func (sb *SmartBootstrapper) bootstrapComponents(
	ctx context.Context,
	cfg app.AdminConfig,
) error {
	logger.Info("Starting smart component bootstrap...")

	// Gather all components to initialize
	components, err := sb.gatherComponents()
	if err != nil {
		return bizerror.Wrap(err, bizerror.UnknownError, "failed to gather components")
	}

	// Sort components by dependencies
	ordered, err := sb.sortComponents(components)
	if err != nil {
		return bizerror.Wrap(err, bizerror.UnknownError, "failed to sort components by dependencies")
	}

	// Initialize components in order
	for i, comp := range ordered {
		logger.Infof("[%d/%d] Initializing %s...", i+1, len(ordered), comp.Type())
		if err := initAndActivateComponent(sb.builder, comp); err != nil {
			return bizerror.Wrap(err, bizerror.UnknownError, fmt.Sprintf("failed to initialize component %s", comp.Type()))
		}
		logger.Infof("[%d/%d] %s initialized successfully", i+1, len(ordered), comp.Type())
	}

	logger.Info("All components bootstrapped successfully")
	return nil
}

// gatherComponents collects all components that need to be initialized
func (sb *SmartBootstrapper) gatherComponents() ([]runtime.Component, error) {
	components := []runtime.Component{}

	// Core components
	coreComps := []struct {
		name   string
		getter func() (runtime.Component, error)
	}{
		{"EventBus", runtime.ComponentRegistry().EventBus},
		{"ResourceStore", runtime.ComponentRegistry().ResourceStore},
		{"ResourceDiscovery", runtime.ComponentRegistry().ResourceDiscovery},
		{"ResourceEngine", runtime.ComponentRegistry().ResourceEngine},
		{"ResourceManager", runtime.ComponentRegistry().ResourceManager},
		{"Console", runtime.ComponentRegistry().Console},
		{"RuleGovernor", runtime.ComponentRegistry().RuleGovernor},
	}

	for _, comp := range coreComps {
		c, err := comp.getter()
		if err != nil {
			return nil, bizerror.Wrap(err, bizerror.UnknownError, fmt.Sprintf("failed to get component %s", comp.name))
		}
		components = append(components, c)
	}

	// Optional components
	optionalComps := []struct {
		name string
		typ  runtime.ComponentType
	}{
		{"CounterManager", counter.ComponentType},
		{"DiagnosticsServer", diagnostics.DiagnosticsServer},
		{"DistributedLock", lock.DistributedLockComponent},
	}

	for _, comp := range optionalComps {
		c, err := runtime.ComponentRegistry().Get(comp.typ)
		if err != nil {
			logger.Warnf("Optional component %s not available: %v", comp.name, err)
			continue
		}
		components = append(components, c)
	}

	return components, nil
}

// sortComponents sorts components by dependency order
func (sb *SmartBootstrapper) sortComponents(
	components []runtime.Component,
) ([]runtime.Component, error) {
	// Build dependency graph and perform topological sort
	graph := runtime.NewDependencyGraph(components)
	sorted, err := graph.TopologicalSort()
	if err != nil {
		return nil, err
	}

	// Log initialization order
	logger.Info("Component initialization order:")
	for i, comp := range sorted {
		deps := comp.RequiredDependencies()
		if len(deps) > 0 {
			logger.Infof("  %d. %s (depends on: %v)", i+1, comp.Type(), deps)
		} else {
			logger.Infof("  %d. %s (no dependencies)", i+1, comp.Type())
		}
	}

	return sorted, nil
}

func initAndActivateComponent(builder *runtime.Builder, comp runtime.Component) error {
	logger.Infof("initializing %s ...", comp.Type())
	if err := comp.Init(builder); err != nil {
		return err
	}
	logger.Infof("%s initialized successfully", comp.Type())
	if err := builder.ActivateComponent(comp); err != nil {
		return bizerror.Wrap(err, bizerror.UnknownError, fmt.Sprintf("failed to activate %s", comp.Type()))
	}
	return nil
}
