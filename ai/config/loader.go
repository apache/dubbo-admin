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

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// mainConfig defines the structure of the main configuration file (internal use only)
type mainConfig struct {
	Project    string         `yaml:"project"`    // Project name
	Version    string         `yaml:"version"`    // Project version
	Components map[string]any `yaml:"components"` // Component configuration path mapping
}

// LoadedConfig contains all loaded configurations
type LoadedConfig struct {
	Project    string
	Version    string
	Components map[string]*Config // Component configurations (key: component name)
}

// Loader handles configuration file loading
type Loader struct {
	configFile string
	configDir  string // Directory containing configuration files
}

// NewLoader creates a new configuration loader
func NewLoader(configFile string) *Loader {
	return &Loader{
		configFile: configFile,
		configDir:  filepath.Dir(configFile),
	}
}

// Load loads and parses all configurations (main entry point)
// Features:
// 1. Load .env file if exists
// 2. Read main configuration file
// 3. Parse environment variables
// 4. Load all component configurations (ordering handled by caller)
func (l *Loader) Load() (*LoadedConfig, error) {
	// 1. Load .env file
	if err := l.loadEnvFile(); err != nil {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	// 2. Read main configuration file
	mainCfg, err := l.loadMainConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load main config: %w", err)
	}

	// 3. Load all component configurations
	components, err := l.loadAllComponents(mainCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load components: %w", err)
	}

	return &LoadedConfig{
		Project:    mainCfg.Project,
		Version:    mainCfg.Version,
		Components: components,
	}, nil
}

// loadEnvFile loads .env file if exists
func (l *Loader) loadEnvFile() error {
	// Load .env file if exists (missing .env is not an error)
	if err := godotenv.Load(); err != nil {
		// File not found is acceptable, only log for other errors
		// os.IsNotExist(err) always returns false for godotenv
		// So we check the error message instead
		if err.Error() != "open .env: no such file or directory" &&
		   err.Error() != "open .env: file does not exist" {
			return err
		}
		// .env file not found is normal, continue execution
	}
	return nil
}

// loadMainConfig loads the main configuration file
func (l *Loader) loadMainConfig() (*mainConfig, error) {
	data, err := os.ReadFile(l.configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg mainConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// loadAllComponents loads all component configurations
func (l *Loader) loadAllComponents(mainCfg *mainConfig) (map[string]*Config, error) {
	components := make(map[string]*Config)

	// Iterate through component declarations in main config
	for name, path := range mainCfg.Components {
		switch v := path.(type) {
		case string:
			// Single configuration file
			cfg, err := l.loadComponent(v)
			if err != nil {
				return nil, fmt.Errorf("failed to load component %s: %w", name, err)
			}
			components[name] = cfg

		case []any:
			// Multiple configuration files (e.g., agents)
			for i, item := range v {
				pathStr, ok := item.(string)
				if !ok {
					continue
				}
				// Use index as name suffix for multiple configs
				componentName := fmt.Sprintf("%s-%d", name, i)
				cfg, err := l.loadComponent(pathStr)
				if err != nil {
					return nil, fmt.Errorf("failed to load component %s: %w", componentName, err)
				}
				components[componentName] = cfg
			}
		}
	}

	return components, nil
}

// loadComponent loads a single component configuration
func (l *Loader) loadComponent(configPath string) (*Config, error) {
	// Resolve relative path
	fullPath := configPath
	if !filepath.IsAbs(configPath) {
		fullPath = filepath.Join(l.configDir, configPath)
	}

	// Read configuration file
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables
	expandedData := os.ExpandEnv(string(data))

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal([]byte(expandedData), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}
