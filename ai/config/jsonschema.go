package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xeipuuv/gojsonschema"
)

// schemaEngine manages the loading, caching, and compilation of JSON schemas.
type schemaEngine struct {
	baseDir  string
	mu       sync.RWMutex
	cache    map[string]*gojsonschema.Schema
	rootObjs map[string]map[string]any
}

// NewSchemaEngine creates a new schemaEngine with the specified base directory.
func NewSchemaEngine(baseDir string) *schemaEngine {
	return &schemaEngine{
		baseDir:  baseDir,
		cache:    make(map[string]*gojsonschema.Schema),
		rootObjs: make(map[string]map[string]any),
	}
}

// ApplyDefaultsAndValidate applies default values and validates.
// This method modifies doc in-place by applying defaults, then validates it.
//
// Parameters:
//   - doc: The configuration document (WILL BE MODIFIED)
//   - schemaFile: The schema filename
//
// Returns:
//   - The modified doc (same map as input)
//   - Error if validation fails
func (e *schemaEngine) ApplyDefaultsAndValidate(doc map[string]any, schemaFile string) (map[string]any, error) {
	compiled, rootObj, err := e.loadSchema(schemaFile)
	if err != nil {
		return nil, err
	}

	// Apply defaults in-place (modifies doc)
	applyDefaults(rootObj, rootObj, doc)

	// Validate the result
	if err := validateJSONSchema(compiled, doc); err != nil {
		return nil, err
	}

	return doc, nil
}

func (e *schemaEngine) loadSchema(fileName string) (*gojsonschema.Schema, map[string]any, error) {
	e.mu.RLock()
	compiled, hasCached := e.cache[fileName]
	rootObj, hasRoot := e.rootObjs[fileName]
	e.mu.RUnlock()

	if hasCached && hasRoot {
		return compiled, rootObj, nil
	}

	fullPath := filepath.Join(e.baseDir, fileName)
	raw, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, nil, fmt.Errorf("structural error: failed to read schema file %s: %w", fileName, err)
	}

	var schemaObj map[string]any
	if err := json.Unmarshal(raw, &schemaObj); err != nil {
		return nil, nil, fmt.Errorf("structural error: failed to parse schema file %s: %w", fileName, err)
	}

	compiled, err = gojsonschema.NewSchema(gojsonschema.NewBytesLoader(raw))
	if err != nil {
		return nil, nil, fmt.Errorf("structural error: failed to compile schema file %s: %w", fileName, err)
	}

	e.mu.Lock()
	e.cache[fileName] = compiled
	e.rootObjs[fileName] = schemaObj
	e.mu.Unlock()
	return compiled, schemaObj, nil
}

func validateJSONSchema(compiled *gojsonschema.Schema, doc any) error {
	docRaw, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal config for schema validation: %w", err)
	}

	result, err := compiled.Validate(gojsonschema.NewBytesLoader(docRaw))
	if err != nil {
		return fmt.Errorf("failed to validate schema: %w", err)
	}
	if result.Valid() {
		return nil
	}

	errMsgs := make([]string, 0, len(result.Errors()))
	for _, e := range result.Errors() {
		field := e.Field()
		if field == "(root)" || field == "" {
			field = "root"
		}
		errMsgs = append(errMsgs, fmt.Sprintf("%s: %s", field, e.Description()))
	}
	return fmt.Errorf("structural error: %s", strings.Join(errMsgs, "; "))
}

// applyDefaults recursively applies default values from schema to value
// Modifies value in-place
func applyDefaults(root map[string]any, schema map[string]any, value any) {
	resolved := resolveSchemaRef(root, schema)

	switch v := value.(type) {
	case map[string]any:
		props, _ := resolved["properties"].(map[string]any)
		for key, propVal := range props {
			propSchema, ok := propVal.(map[string]any)
			if !ok {
				continue
			}
			propSchema = resolveSchemaRef(root, propSchema)

			// Apply default value if property is missing
			if _, exists := v[key]; !exists {
				if defVal, hasDefault := propSchema["default"]; hasDefault {
					v[key] = defVal
				}
			}

			// Recursively apply defaults to nested properties
			if child, exists := v[key]; exists {
				applyDefaults(root, propSchema, child)
			}
		}

	case []any:
		if items, ok := resolved["items"].(map[string]any); ok {
			items = resolveSchemaRef(root, items)
			for i := range v {
				applyDefaults(root, items, v[i])
			}
		}
	}
}

// resolveSchemaRef resolves JSON Pointer references ($ref) within a schema
func resolveSchemaRef(root map[string]any, schema map[string]any) map[string]any {
	if ref, ok := schema["$ref"].(string); ok && strings.HasPrefix(ref, "#/") {
		parts := strings.Split(strings.TrimPrefix(ref, "#/"), "/")
		var cur any = root
		for _, p := range parts {
			obj, ok := cur.(map[string]any)
			if !ok {
				return schema
			}
			next, ok := obj[p]
			if !ok {
				return schema
			}
			cur = next
		}
		if resolved, ok := cur.(map[string]any); ok {
			return resolveSchemaRef(root, resolved)
		}
	}
	return schema
}
