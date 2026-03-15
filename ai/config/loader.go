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
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// LoadedConfig contains all loaded configurations
type LoadedConfig struct {
	Project    string
	Version    string
	SchemaDir  string               // Schema directory path (from environment)
	Components map[string][]*Config // Component configurations grouped by component type
}

// Loader handles configuration file loading
type Loader struct {
	configFile   string
	configDir    string // Directory containing configuration files
	schemaDir    string
	schemaEngine *schemaEngine // Schema validation engine
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
// 2. Initialize schema engine from SCHEMA_DIR (or default schema/json)
// 3. Read and validate main configuration file
// 4. Load all component configurations (ordering handled by caller)
func (l *Loader) Load() (*LoadedConfig, error) {
	// 1. Initialize schema engine
	if err := l.ensureSchemaEngine(); err != nil {
		return nil, err
	}

	// 2. Read, validate, and expand main configuration into loaded component specs.
	loadedCfg, err := l.loadMainConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load main config: %w", err)
	}

	loadedCfg.SchemaDir = l.schemaDir
	return loadedCfg, nil
}

// LoadComponent loads and validates a single component config file.
// It reuses the same structural pipeline as Load():
// yaml.Unmarshal -> schema defaults+validation -> decodeYAMLStrict(KnownFields=true).
func (l *Loader) LoadComponent(configPath string) (*Config, error) {
	if err := l.ensureSchemaEngine(); err != nil {
		return nil, err
	}
	return l.loadComponent(configPath)
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

func (l *Loader) ensureSchemaEngine() error {
	if l.schemaEngine != nil {
		return nil
	}

	if err := l.loadEnvFile(); err != nil {
		return fmt.Errorf("failed to load .env file: %w", err)
	}

	schemaDir, err := l.resolveSchemaDir()
	if err != nil {
		return err
	}

	l.schemaDir = schemaDir
	l.schemaEngine = NewSchemaEngine(schemaDir)
	return nil
}

func (l *Loader) resolveSchemaDir() (string, error) {
	schemaDir := os.Getenv("SCHEMA_DIR")
	if schemaDir == "" {
		schemaDir = filepath.Join(l.configDir, "schema", "json")
	}

	if !filepath.IsAbs(schemaDir) {
		schemaDir = filepath.Join(l.configDir, schemaDir)
	}

	absDir, err := filepath.Abs(schemaDir)
	if err != nil {
		return "", fmt.Errorf("structural error: failed to resolve schema directory: %w", err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return "", fmt.Errorf("structural error: schema directory not found: %s", absDir)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("structural error: schema directory is not a directory: %s", absDir)
	}

	return absDir, nil
}

// loadMainConfig loads and validates the main configuration file, then expands component paths.
func (l *Loader) loadMainConfig() (*LoadedConfig, error) {
	data, err := os.ReadFile(l.configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	normalized, err := l.schemaEngine.ApplyDefaultsAndValidate(raw, "main.schema.json")
	if err != nil {
		return nil, err
	}

	normalizedData, err := yaml.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal normalized main config: %w", err)
	}

	var mainCfg struct {
		Project    string            `yaml:"project"`
		Version    string            `yaml:"version"`
		Components map[string]string `yaml:"components"`
	}

	if err := decodeYAMLStrict(normalizedData, &mainCfg); err != nil {
		return nil, err
	}

	components := make(map[string][]*Config, len(mainCfg.Components))
	for componentType, cfgPath := range mainCfg.Components {
		cfg, err := l.loadComponent(cfgPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load component %s: %w", componentType, err)
		}
		if cfg.Type != componentType {
			return nil, fmt.Errorf("structural error: components.%s points to type %s", componentType, cfg.Type)
		}
		components[componentType] = append(components[componentType], cfg)
	}

	return &LoadedConfig{
		Project:    mainCfg.Project,
		Version:    mainCfg.Version,
		Components: components,
	}, nil
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

	var raw map[string]any
	if err := yaml.Unmarshal([]byte(expandedData), &raw); err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}
	componentType, _ := raw["type"].(string)
	if componentType == "" {
		return nil, fmt.Errorf("structural error: type is required")
	}
	componentSchema, err := schemaFileForComponent(componentType)
	if err != nil {
		return nil, err
	}
	normalized, err := l.schemaEngine.ApplyDefaultsAndValidate(raw, componentSchema)
	if err != nil {
		return nil, err
	}
	normalizedData, err := yaml.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal normalized component config: %w", err)
	}

	// Parse YAML
	var cfg Config
	if err := decodeYAMLStrict(normalizedData, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func schemaFileForComponent(componentType string) (string, error) {
	switch componentType {
	case "logger":
		return "logger.schema.json", nil
	case "memory":
		return "memory.schema.json", nil
	case "models":
		return "models.schema.json", nil
	case "tools":
		return "tools.schema.json", nil
	case "server":
		return "server.schema.json", nil
	case "rag":
		return "rag.schema.json", nil
	case "agent":
		return "agent.schema.json", nil
	default:
		return "", fmt.Errorf("structural error: unsupported component type: %s", componentType)
	}
}

func decodeYAMLStrict(data []byte, target any) error {
	var parserCheck yaml.Node
	if err := yaml.Unmarshal(data, &parserCheck); err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(target); err != nil {
		return fmt.Errorf("structural error: %w", err)
	}

	return nil
}
