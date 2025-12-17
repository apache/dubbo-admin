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
	"github.com/duke-git/lancet/v2/slice"

	"github.com/pkg/errors"

	"github.com/apache/dubbo-admin/pkg/config/app"
	"github.com/apache/dubbo-admin/pkg/console/counter"
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

	// Optional: Set bootstrap mode based on configuration
	// bootstrapper.SetMode(StrictMode) // Uncomment for strict dependency checking

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

// BootstrapMode defines how components are initialized
type BootstrapMode int

const (
	// CompatibleMode uses dependency declarations when available, falls back to Order()
	// This is the default mode for smooth migration
	CompatibleMode BootstrapMode = iota

	// StrictMode requires all components to declare dependencies, ignores Order()
	// Use this mode for new development or after all components are migrated
	StrictMode
)

// SmartBootstrapper handles intelligent component initialization
type SmartBootstrapper struct {
	builder *runtime.Builder
	mode    BootstrapMode
}

// NewSmartBootstrapper creates a new smart bootstrapper
func NewSmartBootstrapper(builder *runtime.Builder) *SmartBootstrapper {
	return &SmartBootstrapper{
		builder: builder,
		mode:    CompatibleMode, // Default to compatible mode
	}
}

// SetMode changes the bootstrap mode
func (sb *SmartBootstrapper) SetMode(mode BootstrapMode) {
	sb.mode = mode
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
		return errors.Wrap(err, "failed to gather components")
	}

	// Sort components by dependencies
	ordered, err := sb.sortComponents(components)
	if err != nil {
		return errors.Wrap(err, "failed to sort components by dependencies")
	}

	// Initialize components in order
	for i, comp := range ordered {
		logger.Infof("[%d/%d] Initializing %s...", i+1, len(ordered), comp.Type())
		if err := initAndActivateComponent(sb.builder, comp); err != nil {
			return errors.Wrapf(err, "failed to initialize component %s", comp.Type())
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
	}

	for _, comp := range coreComps {
		c, err := comp.getter()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get component %s", comp.name)
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
	// Categorize components
	withDeps := []runtime.ComponentWithDependencies{}
	withoutDeps := []runtime.Component{}

	for _, comp := range components {
		if dep, ok := comp.(runtime.ComponentWithDependencies); ok {
			withDeps = append(withDeps, dep)
			logger.Debugf("Component %s declares dependencies: %v",
				comp.Type(), dep.RequiredDependencies())
		} else {
			withoutDeps = append(withoutDeps, comp)
			logger.Debugf("Component %s uses Order() mode: %d",
				comp.Type(), comp.Order())
		}
	}

	// If no components declare dependencies, fall back to Order() sorting
	if len(withDeps) == 0 {
		logger.Info("No components declare dependencies, using Order() based initialization")
		return sortByOrder(components), nil
	}

	// In strict mode, all components must declare dependencies
	if sb.mode == StrictMode && len(withoutDeps) > 0 {
		names := []string{}
		for _, comp := range withoutDeps {
			names = append(names, string(comp.Type()))
		}
		return nil, fmt.Errorf(
			"strict mode enabled but the following components don't declare dependencies: %v\n"+
				"Please implement RequiredDependencies() for these components",
			names,
		)
	}

	// Build dependency graph and perform topological sort
	graph := runtime.NewDependencyGraph(components)
	sorted, err := graph.TopologicalSort()
	if err != nil {
		return nil, err
	}

	// Log initialization order
	logger.Info("Component initialization order:")
	for i, comp := range sorted {
		logger.Infof("  %d. %s", i+1, comp.Type())
	}

	return sorted, nil
}

// sortByOrder sorts components by Order() value (legacy mode)
func sortByOrder(components []runtime.Component) []runtime.Component {
	sorted := make([]runtime.Component, len(components))
	copy(sorted, components)

	slice.SortBy(sorted, func(a, b runtime.Component) bool {
		return a.Order() > b.Order()
	})

	return sorted
}

// Legacy functions below are kept for reference but no longer used
// They can be removed in a future version after all components are migrated

func initEventBus(builder *runtime.Builder) error {
	comp, err := runtime.ComponentRegistry().EventBus()
	if err != nil {
		return err
	}
	return initAndActivateComponent(builder, comp)
}

func initResourceStore(cfg app.AdminConfig, builder *runtime.Builder) error {
	comp, err := runtime.ComponentRegistry().ResourceStore()
	if err != nil {
		return errors.Wrapf(err, "could not retrieve resource store %s component", cfg.Store.Type)
	}
	return initAndActivateComponent(builder, comp)
}
func initResourceManager(builder *runtime.Builder) error {
	comp, err := runtime.ComponentRegistry().ResourceManager()
	if err != nil {
		return err
	}
	return initAndActivateComponent(builder, comp)
}

func initializeConsole(builder *runtime.Builder) error {
	comp, err := runtime.ComponentRegistry().Console()
	if err != nil {
		return err
	}
	return initAndActivateComponent(builder, comp)
}

func initializeResourceDiscovery(builder *runtime.Builder) error {
	comp, err := runtime.ComponentRegistry().ResourceDiscovery()
	if err != nil {
		return err
	}
	return initAndActivateComponent(builder, comp)
}

func initializeResourceEngine(builder *runtime.Builder) error {
	comp, err := runtime.ComponentRegistry().ResourceEngine()
	if err != nil {
		return err
	}
	return initAndActivateComponent(builder, comp)
}

func initializeDiagnoticsServer(builder *runtime.Builder) error {
	comp, err := runtime.ComponentRegistry().Get(diagnostics.DiagnosticsServer)
	if err != nil {
		return err
	}
	return initAndActivateComponent(builder, comp)
}

func initializeCounterManager(builder *runtime.Builder) error {
	comp, err := runtime.ComponentRegistry().Get(counter.ComponentType)
	if err != nil {
		return err
	}
	return initAndActivateComponent(builder, comp)
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
