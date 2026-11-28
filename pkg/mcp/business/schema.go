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

package business

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/jsonschema-go/jsonschema"
	"google.golang.org/protobuf/types/known/structpb"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
)

const jsonSchema202012 = "https://json-schema.org/draft/2020-12/schema"

type schemaBuilder struct {
	types    map[string]*meshproto.MCPTypeDefinition
	defs     map[string]*jsonschema.Schema
	building map[string]bool
}

func BuildInputSchema(
	parameterTypes []string,
	bindings []*meshproto.MCPParameterBinding,
	types map[string]*meshproto.MCPTypeDefinition,
) (*structpb.Struct, error) {
	if err := ValidateParameterBindings(bindings, parameterTypes); err != nil {
		return nil, err
	}
	builder := &schemaBuilder{
		types:    types,
		defs:     make(map[string]*jsonschema.Schema),
		building: make(map[string]bool),
	}
	root := &jsonschema.Schema{
		Schema:               jsonSchema202012,
		Type:                 "object",
		Properties:           make(map[string]*jsonschema.Schema, len(bindings)),
		AdditionalProperties: denySchema(),
		Required:             make([]string, len(bindings)),
		PropertyOrder:        make([]string, len(bindings)),
	}
	for _, binding := range bindings {
		position := int(binding.Position)
		property, err := builder.schemaForType(parameterTypes[position])
		if err != nil {
			return nil, fmt.Errorf("parameter %q: %w", binding.Alias, err)
		}
		property.Description = binding.Description
		root.Properties[binding.Alias] = property
		root.Required[position] = binding.Alias
		root.PropertyOrder[position] = binding.Alias
	}
	root.Defs = builder.defs
	if _, err := root.Resolve(nil); err != nil {
		return nil, fmt.Errorf("resolve input schema: %w", err)
	}
	data, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		return nil, err
	}
	return structpb.NewStruct(object)
}

func CompileInputSchema(schema *structpb.Struct) (*jsonschema.Resolved, error) {
	if schema == nil {
		return nil, fmt.Errorf("input schema is required")
	}
	data, err := schema.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var parsed jsonschema.Schema
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}
	resolved, err := parsed.Resolve(nil)
	if err != nil {
		return nil, fmt.Errorf("resolve input schema: %w", err)
	}
	return resolved, nil
}

func (b *schemaBuilder) schemaForType(typeName string) (*jsonschema.Schema, error) {
	if schema, known, err := primitiveSchema(typeName); known || err != nil {
		return schema, err
	}
	if isUnsupportedType(typeName) {
		return nil, fmt.Errorf("type %q is not supported", typeName)
	}
	if _, exists := b.types[typeName]; !exists {
		return nil, fmt.Errorf("type definition %q is missing", typeName)
	}
	if err := b.buildDefinition(typeName); err != nil {
		return nil, err
	}
	return nullableSchema(&jsonschema.Schema{Ref: "#/$defs/" + definitionKey(typeName)}), nil
}

func (b *schemaBuilder) buildDefinition(typeName string) error {
	key := definitionKey(typeName)
	if _, exists := b.defs[key]; exists {
		return nil
	}
	definition := b.types[typeName]
	placeholder := &jsonschema.Schema{}
	b.defs[key] = placeholder
	if b.building[typeName] {
		return nil
	}
	b.building[typeName] = true
	defer delete(b.building, typeName)

	switch {
	case len(definition.Enums) > 0:
		placeholder.Type = "string"
		placeholder.Enum = make([]any, len(definition.Enums))
		for i, value := range definition.Enums {
			placeholder.Enum[i] = value
		}
	case len(definition.Items) == 1:
		itemSchema, err := b.schemaForType(definition.Items[0])
		if err != nil {
			return fmt.Errorf("type %q item: %w", typeName, err)
		}
		placeholder.Type = "array"
		placeholder.Items = itemSchema
	case len(definition.Items) == 2:
		if !isStringType(definition.Items[0]) {
			return fmt.Errorf("map type %q has non-string key %q", typeName, definition.Items[0])
		}
		valueSchema, err := b.schemaForType(definition.Items[1])
		if err != nil {
			return fmt.Errorf("map type %q value: %w", typeName, err)
		}
		placeholder.Type = "object"
		placeholder.AdditionalProperties = valueSchema
	case len(definition.Items) > 2:
		return fmt.Errorf("type %q has invalid item arity %d", typeName, len(definition.Items))
	case definition.Properties != nil:
		placeholder.Type = "object"
		placeholder.Properties = make(map[string]*jsonschema.Schema, len(definition.Properties))
		placeholder.AdditionalProperties = denySchema()
		for propertyName, propertyType := range definition.Properties {
			propertySchema, err := b.schemaForType(propertyType)
			if err != nil {
				return fmt.Errorf("type %q property %q: %w", typeName, propertyName, err)
			}
			placeholder.Properties[propertyName] = propertySchema
		}
	default:
		return fmt.Errorf("type %q is open or unresolved", typeName)
	}
	return nil
}

func primitiveSchema(typeName string) (*jsonschema.Schema, bool, error) {
	switch typeName {
	case "boolean":
		return &jsonschema.Schema{Type: "boolean"}, true, nil
	case "java.lang.Boolean":
		return nullableSchema(&jsonschema.Schema{Type: "boolean"}), true, nil
	case "byte":
		return integerSchema(-128, 127, false), true, nil
	case "java.lang.Byte":
		return integerSchema(-128, 127, true), true, nil
	case "short":
		return integerSchema(-32768, 32767, false), true, nil
	case "java.lang.Short":
		return integerSchema(-32768, 32767, true), true, nil
	case "int":
		return integerSchema(-2147483648, 2147483647, false), true, nil
	case "java.lang.Integer":
		return integerSchema(-2147483648, 2147483647, true), true, nil
	case "long":
		return longSchema(false), true, nil
	case "java.lang.Long":
		return longSchema(true), true, nil
	case "float", "double":
		return &jsonschema.Schema{Type: "number"}, true, nil
	case "java.lang.Float", "java.lang.Double":
		return nullableSchema(&jsonschema.Schema{Type: "number"}), true, nil
	case "char":
		return charSchema(false), true, nil
	case "java.lang.Character":
		return charSchema(true), true, nil
	case "java.lang.String", "String", "string":
		return nullableSchema(&jsonschema.Schema{Type: "string"}), true, nil
	case "void", "java.lang.Void":
		return nil, true, fmt.Errorf("void is not a valid parameter type")
	default:
		return nil, false, nil
	}
}

func integerSchema(minimum, maximum float64, nullable bool) *jsonschema.Schema {
	schema := &jsonschema.Schema{Type: "integer", Minimum: &minimum, Maximum: &maximum}
	if nullable {
		return nullableSchema(schema)
	}
	return schema
}

func longSchema(nullable bool) *jsonschema.Schema {
	schema := &jsonschema.Schema{Type: "string", Pattern: "^-?(0|[1-9][0-9]*)$"}
	if nullable {
		return nullableSchema(schema)
	}
	return schema
}

func charSchema(nullable bool) *jsonschema.Schema {
	one := 1
	schema := &jsonschema.Schema{Type: "string", MinLength: &one, MaxLength: &one}
	if nullable {
		return nullableSchema(schema)
	}
	return schema
}

func nullableSchema(schema *jsonschema.Schema) *jsonschema.Schema {
	return &jsonschema.Schema{AnyOf: []*jsonschema.Schema{schema, &jsonschema.Schema{Type: "null"}}}
}

func denySchema() *jsonschema.Schema {
	return &jsonschema.Schema{Not: &jsonschema.Schema{}}
}

func definitionKey(typeName string) string {
	var sanitized strings.Builder
	for _, r := range typeName {
		if r < utf8.RuneSelf && ((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '.' || r == '-') {
			sanitized.WriteRune(r)
		} else {
			sanitized.WriteByte('_')
		}
	}
	digest := sha256.Sum256([]byte(typeName))
	return sanitized.String() + "_" + hex.EncodeToString(digest[:6])
}

func isPrimitiveOrVoid(typeName string) bool {
	_, known, _ := primitiveSchema(typeName)
	return known
}

func isStringType(typeName string) bool {
	return typeName == "java.lang.String" || typeName == "String" || typeName == "string"
}

func isUnsupportedType(typeName string) bool {
	if typeName == "java.lang.Object" || typeName == "Object" || strings.Contains(typeName, "?") {
		return true
	}
	switch typeName {
	case "java.util.List", "java.util.Collection", "java.util.Set", "java.util.Map",
		"List", "Collection", "Set", "Map":
		return true
	}
	unsupported := []string{
		"java.util.Date", "java.time.", "java.math.BigInteger", "java.math.BigDecimal",
	}
	for _, prefix := range unsupported {
		if typeName == prefix || strings.HasPrefix(typeName, prefix) {
			return true
		}
	}
	return false
}
