package runtime

import (
	"context"
	"dubbo-admin-ai/config"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/firebase/genkit/go/genkit"
	"gopkg.in/yaml.v3"
)

// Component defines the interface for all components
type Component interface {
	Name() string
	Validate() error
	Init(*Runtime) error
	Start() error
	Stop() error
}

type instanceNameSetter interface {
	SetName(name string)
}

// ComponentFactory is the function type for creating components
type ComponentFactory func(config *yaml.Node) (Component, error)

var (
	gloRuntime *Runtime = nil
)

type componentEntry struct {
	componentType string
	names         []string
	components    map[string]Component
}

type ComponentSnapshotEntry struct {
	Type  string   `json:"type"`
	Count int      `json:"count"`
	Names []string `json:"names"`
}

func (e *componentEntry) snapshot() ComponentSnapshotEntry {
	if e == nil {
		return ComponentSnapshotEntry{}
	}
	names := make([]string, len(e.names))
	copy(names, e.names)
	return ComponentSnapshotEntry{
		Type:  e.componentType,
		Count: len(e.names),
		Names: names,
	}
}

func (e ComponentSnapshotEntry) String() string {
	return fmt.Sprintf("%s (%d): %s", e.Type, e.Count, strings.Join(e.Names, ", "))
}

func NewRuntime() *Runtime {
	return &Runtime{
		factories:            make(map[string]ComponentFactory),
		orderedComponentType: make([]string, 0),
		genkitOptions:        make([]genkit.GenkitOption, 0),
		componentEntries:     make(map[string]*componentEntry),
	}
}

func Bootstrap(configFile string, registerFactory func(rt *Runtime)) (*Runtime, error) {
	gloRuntime = NewRuntime()

	// Register component factories
	if registerFactory != nil {
		registerFactory(gloRuntime)
	}

	// Create config loader and load all configurations
	loader := config.NewLoader(configFile)
	loadedCfg, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Resolve component entries from loaded configuration and registered factories.
	if err := gloRuntime.createComponents(loadedCfg); err != nil {
		return nil, fmt.Errorf("failed to create components: %w", err)
	}

	// Initialize components in dependency order, which is the order of factory registration.
	for _, componentType := range gloRuntime.orderedComponentType {
		configs := loadedCfg.Components[componentType]
		for _, cfg := range configs {
			comp, err := gloRuntime.createComponent(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to create %s: %w", componentType, err)
			}
			instanceName := gloRuntime.nextComponentName(componentType)
			injectComponentName(comp, instanceName)
			if err := comp.Validate(); err != nil {
				return nil, fmt.Errorf("failed to validate %s: %w", componentType, err)
			}

			if err := comp.Init(gloRuntime); err != nil {
				return nil, fmt.Errorf("failed to init %s: %w", componentType, err)
			}
			gloRuntime.registerComponent(componentType, instanceName, comp)
		}
	}

	// Start all loaded components once after successful registration.
	for _, componentType := range gloRuntime.orderedComponentType {
		entry := gloRuntime.componentEntries[componentType]
		if entry == nil {
			continue
		}
		for _, name := range entry.names {
			comp, err := gloRuntime.GetComponentByName(name)
			if err != nil {
				return nil, fmt.Errorf("failed to get %s for startup: %w", name, err)
			}
			if err := comp.Start(); err != nil {
				return nil, fmt.Errorf("failed to start %s: %w", name, err)
			}
		}
	}
	snapshot := gloRuntime.ComponentSnapshot()
	gloRuntime.GetLogger().Info("Runtime component snapshot", "components", snapshot)
	return gloRuntime, nil
}

type Runtime struct {
	mu             sync.RWMutex
	configFile     string
	genkitRegistry *genkit.Genkit
	genkitOptions  []genkit.GenkitOption
	factories      map[string]ComponentFactory

	orderedComponentType []string
	componentEntries     map[string]*componentEntry
}

// RegisterFactory registers a component factory function
func (r *Runtime) RegisterFactory(componentType string, factory ComponentFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[componentType]; exists {
		fmt.Printf("Warning: component type '%s' is already registered, overwriting\n", componentType)
	} else {
		// Only record order on first registration
		r.orderedComponentType = append(r.orderedComponentType, componentType)
		r.componentEntries[componentType] = &componentEntry{
			componentType: componentType,
			names:         make([]string, 0),
			components:    make(map[string]Component),
		}
	}

	r.factories[componentType] = factory
}

// RegisterGenkitOption registers Genkit initialization options
func (r *Runtime) RegisterGenkitOption(opts ...genkit.GenkitOption) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.genkitOptions = append(r.genkitOptions, opts...)
}

func (r *Runtime) SetGenkitRegistry(registry *genkit.Genkit) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.genkitRegistry = registry
}

func (r *Runtime) GetFactoryFn(componentType string) (ComponentFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, exists := r.factories[componentType]
	if !exists {
		return nil, fmt.Errorf("component type '%s' not registered", componentType)
	}

	return factory, nil
}

// createComponents validates loaded components and assigns them into component entries.
func (r *Runtime) createComponents(loadedCfg *config.LoadedConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, entry := range r.componentEntries {
		entry.names = entry.names[:0]
		clear(entry.components)
	}

	for _, componentType := range r.orderedComponentType {
		entry := r.componentEntries[componentType]
		if entry == nil {
			return fmt.Errorf("component entry not found for type %s", componentType)
		}
	}

	for componentType := range loadedCfg.Components {
		if _, exists := r.factories[componentType]; !exists {
			return fmt.Errorf("no factory for %s", componentType)
		}
	}

	return nil
}

// createComponent get factory by component type and create component instance
func (r *Runtime) createComponent(cfg *config.Config) (Component, error) {
	factoryFn, err := r.GetFactoryFn(cfg.Type)
	if err != nil {
		return nil, fmt.Errorf("no factory for %s: %w", cfg.Type, err)
	}

	comp, err := factoryFn(&cfg.Spec)
	if err != nil {
		return nil, fmt.Errorf("factory failed for %s: %w", cfg.Type, err)
	}

	return comp, nil
}

func GetLogger() *slog.Logger {
	if gloRuntime == nil {
		return slog.Default()
	}
	return gloRuntime.GetLogger()
}

func (rt *Runtime) GetLogger() *slog.Logger {
	return slog.Default()
}

func (rt *Runtime) GetContext() context.Context {
	return context.Background()
}

func (rt *Runtime) GetRegistry() *genkit.Genkit {
	if rt.genkitRegistry == nil {
		panic("Genkit registry not initialized")
	}
	return rt.genkitRegistry
}

func (rt *Runtime) GetGenkitRegistry() *genkit.Genkit {
	// Returns nil before genkit.Init() is called, does not panic
	// Components must check if return value is nil
	return rt.genkitRegistry
}

// GetComponent retrieves a single component by type first, then by instance name.
func (rt *Runtime) GetComponent(name string) (Component, error) {
	components, err := rt.GetComponentByType(name)
	if err == nil {
		if len(components) == 1 {
			return components[0], nil
		}
		return nil, fmt.Errorf("multiple components found for type %s", name)
	}

	return rt.GetComponentByName(name)
}

// GetComponentByType retrieves all component instances for a component type.
func (rt *Runtime) GetComponentByType(componentType string) ([]Component, error) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	entry, ok := rt.componentEntries[componentType]
	if !ok || len(entry.names) == 0 {
		return nil, fmt.Errorf("component type not found: %s", componentType)
	}

	components := make([]Component, 0, len(entry.names))
	for _, name := range entry.names {
		components = append(components, entry.components[name])
	}
	return components, nil
}

// GetComponentByName retrieves a component instance by generated instance name.
func (rt *Runtime) GetComponentByName(name string) (Component, error) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	for _, componentType := range rt.orderedComponentType {
		entry := rt.componentEntries[componentType]
		if entry == nil {
			continue
		}
		comp, ok := entry.components[name]
		if ok {
			return comp, nil
		}
	}

	return nil, fmt.Errorf("component not found: %s", name)
}

func (rt *Runtime) RegisterComponent(comp Component) {
	componentType := comp.Name()
	instanceName := rt.nextComponentName(componentType)
	injectComponentName(comp, instanceName)
	rt.registerComponent(componentType, instanceName, comp)
}

// StopAll stops registered components in reverse registration order.
func (rt *Runtime) StopAll() error {
	rt.mu.RLock()
	orderedTypes := make([]string, len(rt.orderedComponentType))
	copy(orderedTypes, rt.orderedComponentType)

	namesByType := make(map[string][]string, len(rt.componentEntries))
	for componentType, entry := range rt.componentEntries {
		if entry == nil {
			continue
		}
		names := make([]string, len(entry.names))
		copy(names, entry.names)
		namesByType[componentType] = names
	}
	rt.mu.RUnlock()

	for i := len(orderedTypes) - 1; i >= 0; i-- {
		componentType := orderedTypes[i]
		names := namesByType[componentType]
		for j := len(names) - 1; j >= 0; j-- {
			name := names[j]
			comp, err := rt.GetComponentByName(name)
			if err != nil {
				return fmt.Errorf("failed to get %s for shutdown: %w", name, err)
			}
			if err := comp.Stop(); err != nil {
				return fmt.Errorf("failed to stop %s: %w", name, err)
			}
		}
	}

	return nil
}

// ComponentSnapshot returns a stable structured snapshot of registered components.
func (rt *Runtime) ComponentSnapshot() []ComponentSnapshotEntry {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	entries := make([]ComponentSnapshotEntry, 0, len(rt.orderedComponentType))
	for _, componentType := range rt.orderedComponentType {
		entry := rt.componentEntries[componentType]
		if entry == nil {
			continue
		}
		entries = append(entries, entry.snapshot())
	}

	return entries
}

func FormatComponentSnapshot(entries []ComponentSnapshotEntry) string {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, "- "+entry.String())
	}
	return strings.Join(lines, "\n")
}

func (rt *Runtime) nextComponentName(componentType string) string {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	entry := rt.componentEntries[componentType]
	if entry == nil {
		return fmt.Sprintf("%s-0", componentType)
	}
	return fmt.Sprintf("%s-%d", componentType, len(entry.names))
}

func (rt *Runtime) registerComponent(componentType, name string, comp Component) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	entry, ok := rt.componentEntries[componentType]
	if !ok {
		entry = &componentEntry{
			componentType: componentType,
			names:         make([]string, 0),
			components:    make(map[string]Component),
		}
		rt.componentEntries[componentType] = entry
		rt.orderedComponentType = append(rt.orderedComponentType, componentType)
	}

	entry.names = append(entry.names, name)
	entry.components[name] = comp
}

func injectComponentName(comp Component, name string) {
	if setter, ok := comp.(instanceNameSetter); ok {
		setter.SetName(name)
	}
}

func GetRuntime() *Runtime {
	if gloRuntime == nil {
		panic("Runtime not initialized, call Bootstrap() first")
	}
	return gloRuntime
}
