package runtime

import (
	"context"
	"dubbo-admin-ai/config"
	"fmt"
	"log/slog"
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

// ComponentFactory is the function type for creating components
type ComponentFactory func(config *yaml.Node) (Component, error)

var (
	gloRuntime *Runtime = nil
)

func NewRuntime() *Runtime {
	return &Runtime{
		factories:     make(map[string]ComponentFactory),
		factoryOrder:  make([]string, 0),
		genkitOptions: make([]genkit.GenkitOption, 0),
	}
}

func Bootstrap(configFile string, registerFn func(rt *Runtime)) (*Runtime, error) {
	gloRuntime = NewRuntime()

	// Register component factories
	if registerFn != nil {
		registerFn(gloRuntime)
	}

	// Create config loader and load all configurations
	loader := config.NewLoader(configFile)
	loadedCfg, err := loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Create component instances
	instances, err := gloRuntime.createComponents(loadedCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create components: %w", err)
	}

	// Initialize components in dependency order, which is the order of factory registration.
	for _, comp := range instances {
		if err := comp.Validate(); err != nil {
			return nil, fmt.Errorf("failed to validate %s: %w", comp.Name(), err)
		}

		if err := comp.Init(gloRuntime); err != nil {
			return nil, fmt.Errorf("failed to init %s: %w", comp.Name(), err)
		}
		gloRuntime.Components.Store(comp.Name(), comp)
	}

	// Start all loaded components
	gloRuntime.Components.Range(func(key, value any) bool {
		comp := value.(Component)
		if err := comp.Start(); err != nil {
			return false
		}
		return true
	})

	return gloRuntime, nil
}

type Runtime struct {
	mu             sync.RWMutex
	configFile     string
	genkitRegistry *genkit.Genkit
	genkitOptions  []genkit.GenkitOption
	factories      map[string]ComponentFactory
	factoryOrder   []string

	Components sync.Map
}

// RegisterFactory registers a component factory function
func (r *Runtime) RegisterFactory(componentType string, factory ComponentFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[componentType]; exists {
		fmt.Printf("Warning: component type '%s' is already registered, overwriting\n", componentType)
	} else {
		// Only record order on first registration
		r.factoryOrder = append(r.factoryOrder, componentType)
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

// Creates component instances based on loaded configuration and factory registration order
func (r *Runtime) createComponents(loadedCfg *config.LoadedConfig) ([]Component, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var instances []Component
	processed := make(map[string]bool)

	// Create components following factory registration order
	for _, componentType := range r.factoryOrder {
		// Find and create the component with this type
		for name, cfg := range loadedCfg.Components {
			if processed[name] {
				continue
			}

			if cfg.Type == componentType {
				comp, err := r.createComponent(cfg)
				if err != nil {
					return nil, fmt.Errorf("failed to create %s: %w", name, err)
				}

				instances = append(instances, comp)
				processed[name] = true
			}
		}
	}

	// Fail fast when configuration contains component types without registered factories.
	for name, cfg := range loadedCfg.Components {
		if processed[name] {
			continue
		}
		if _, exists := r.factories[cfg.Type]; !exists {
			return nil, fmt.Errorf("no factory for %s", cfg.Type)
		}
	}

	return instances, nil
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

// GetComponent retrieves a component instance by name
func (rt *Runtime) GetComponent(name string) (Component, error) {
	v, ok := rt.Components.Load(name)
	if !ok {
		return nil, fmt.Errorf("component not found: %s", name)
	}

	return v.(Component), nil
}

func (rt *Runtime) RegisterComponent(comp Component) {
	rt.Components.Store(comp.Name(), comp)
}

func GetRuntime() *Runtime {
	if gloRuntime == nil {
		panic("Runtime not initialized, call Bootstrap() first")
	}
	return gloRuntime
}
