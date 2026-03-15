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
	applyDefaults(rootObj, rootObj, doc, nil)

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

// applyDefaults recursively applies default values from schema to value.
// Modifies value in-place.
func applyDefaults(root map[string]any, schema map[string]any, value any, parent map[string]any) {
	resolved := resolveSchemaRef(root, schema)
	for _, branch := range expandCompositeSchemas(root, resolved, value, parent) {
		applyDefaults(root, branch, value, parent)
	}

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
					v[key] = cloneJSONValue(defVal)
				}
			}

			// Recursively apply defaults to nested properties
			if child, exists := v[key]; exists {
				applyDefaults(root, propSchema, child, v)
			}
		}

	case []any:
		if items, ok := resolved["items"].(map[string]any); ok {
			items = resolveSchemaRef(root, items)
			for i := range v {
				applyDefaults(root, items, v[i], nil)
			}
		}
	}
}

func expandCompositeSchemas(root map[string]any, schema map[string]any, value any, parent map[string]any) []map[string]any {
	branches := make([]map[string]any, 0)

	if allOf, ok := schema["allOf"].([]any); ok {
		for _, branch := range allOf {
			branchSchema, ok := branch.(map[string]any)
			if !ok {
				continue
			}
			branches = append(branches, resolveSchemaRef(root, branchSchema))
		}
	}

	if branch := selectCompositeBranch(root, schema, value, parent, "oneOf"); branch != nil {
		branches = append(branches, branch)
	}
	if branch := selectCompositeBranch(root, schema, value, parent, "anyOf"); branch != nil {
		branches = append(branches, branch)
	}

	return branches
}

func selectCompositeBranch(root map[string]any, schema map[string]any, value any, parent map[string]any, keyword string) map[string]any {
	rawBranches, ok := schema[keyword].([]any)
	if !ok {
		return nil
	}

	discriminator := ""
	if parent != nil {
		discriminator, _ = parent["type"].(string)
	}

	var firstCompatible map[string]any
	for _, raw := range rawBranches {
		branchSchema, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if discriminator != "" {
			if ref, ok := branchSchema["$ref"].(string); ok && strings.Contains(ref, discriminator) {
				resolved := resolveSchemaRef(root, branchSchema)
				if isSchemaCompatible(resolved, value) {
					return resolved
				}
			}
		}

		resolved := resolveSchemaRef(root, branchSchema)
		if isSchemaCompatible(resolved, value) && firstCompatible == nil {
			firstCompatible = resolved
		}
	}

	return firstCompatible
}

func isSchemaCompatible(schema map[string]any, value any) bool {
	if schemaType, ok := schema["type"].(string); ok && !matchesSchemaType(schemaType, value) {
		return false
	}

	if constVal, ok := schema["const"]; ok {
		return value == constVal
	}

	if enumVals, ok := schema["enum"].([]any); ok && value != nil {
		for _, enumVal := range enumVals {
			if value == enumVal {
				return true
			}
		}
		return false
	}

	obj, ok := value.(map[string]any)
	if !ok {
		return true
	}

	props, _ := schema["properties"].(map[string]any)
	additional, hasAdditional := schema["additionalProperties"]
	for key := range obj {
		if _, exists := props[key]; exists {
			continue
		}
		if !hasAdditional {
			continue
		}
		if allow, ok := additional.(bool); ok && !allow {
			return false
		}
	}

	return true
}

func matchesSchemaType(schemaType string, value any) bool {
	switch schemaType {
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "integer":
		switch value.(type) {
		case int, int8, int16, int32, int64:
			return true
		case uint, uint8, uint16, uint32, uint64:
			return true
		case float64:
			return value == float64(int64(value.(float64)))
		default:
			return false
		}
	case "number":
		switch value.(type) {
		case int, int8, int16, int32, int64:
			return true
		case uint, uint8, uint16, uint32, uint64:
			return true
		case float32, float64:
			return true
		default:
			return false
		}
	case "null":
		return value == nil
	default:
		return true
	}
}

func cloneJSONValue(v any) any {
	switch vv := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(vv))
		for key, val := range vv {
			out[key] = cloneJSONValue(val)
		}
		return out
	case []any:
		out := make([]any, len(vv))
		for i := range vv {
			out[i] = cloneJSONValue(vv[i])
		}
		return out
	default:
		return v
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
