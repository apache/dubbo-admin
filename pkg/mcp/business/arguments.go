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
	"encoding/json"
	"fmt"
	"strconv"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/util/genericinvoke"
)

func RestoreArguments(contract *ResolvedContract, arguments map[string]any) ([]any, error) {
	if contract == nil || contract.Operation == nil {
		return nil, fmt.Errorf("resolved contract is required")
	}
	if err := ValidateParameterBindings(contract.Parameters, contract.Operation.ParameterTypes); err != nil {
		return nil, err
	}
	validator, err := CompileInputSchema(contract.InputSchema)
	if err != nil {
		return nil, err
	}
	if err := validator.Validate(arguments); err != nil {
		return nil, fmt.Errorf("arguments do not match input schema: %w", err)
	}

	types := make(map[string]*meshproto.MCPTypeDefinition, len(contract.RelatedTypes))
	for _, typ := range contract.RelatedTypes {
		if typ != nil {
			types[typ.Type] = typ
		}
	}
	result := make([]any, len(contract.Operation.ParameterTypes))
	for _, binding := range contract.Parameters {
		value, exists := arguments[binding.Alias]
		if !exists {
			return nil, fmt.Errorf("required argument %q is missing", binding.Alias)
		}
		position := int(binding.Position)
		decoded, err := decodeValue(contract.Operation.ParameterTypes[position], value, types)
		if err != nil {
			return nil, fmt.Errorf("argument %q: %w", binding.Alias, err)
		}
		result[position] = decoded
	}
	return result, nil
}

func decodeValue(typeName string, value any, types map[string]*meshproto.MCPTypeDefinition) (any, error) {
	if value == nil {
		if isNullableType(typeName) {
			return nil, nil
		}
		return nil, fmt.Errorf("null is not valid for %s", typeName)
	}
	if isScalarType(typeName) {
		if typeName == "long" || typeName == "java.lang.Long" {
			text, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("expected decimal string for %s", typeName)
			}
			if _, err := strconv.ParseInt(text, 10, 64); err != nil {
				return nil, fmt.Errorf("invalid %s value: %w", typeName, err)
			}
			return text, nil
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		return genericinvoke.DecodeValue(typeName, raw)
	}

	definition, exists := types[typeName]
	if !exists {
		return nil, fmt.Errorf("type definition %q is missing", typeName)
	}
	switch {
	case len(definition.Enums) > 0:
		return value, nil
	case len(definition.Items) == 1:
		items, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("expected array for %s", typeName)
		}
		decoded := make([]any, len(items))
		for i, item := range items {
			converted, err := decodeValue(definition.Items[0], item, types)
			if err != nil {
				return nil, fmt.Errorf("item %d: %w", i, err)
			}
			decoded[i] = converted
		}
		return decoded, nil
	case len(definition.Items) == 2:
		object, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected object for %s", typeName)
		}
		decoded := make(map[string]any, len(object))
		for key, item := range object {
			converted, err := decodeValue(definition.Items[1], item, types)
			if err != nil {
				return nil, fmt.Errorf("map value %q: %w", key, err)
			}
			decoded[key] = converted
		}
		return decoded, nil
	case definition.Properties != nil:
		object, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected object for %s", typeName)
		}
		decoded := make(map[string]any, len(object))
		for key, item := range object {
			propertyType, exists := definition.Properties[key]
			if !exists {
				return nil, fmt.Errorf("unknown property %q", key)
			}
			converted, err := decodeValue(propertyType, item, types)
			if err != nil {
				return nil, fmt.Errorf("property %q: %w", key, err)
			}
			decoded[key] = converted
		}
		return decoded, nil
	default:
		return nil, fmt.Errorf("type %q is open or unresolved", typeName)
	}
}

func isScalarType(typeName string) bool {
	_, known, _ := primitiveSchema(typeName)
	return known && typeName != "void" && typeName != "java.lang.Void"
}

func isNullableType(typeName string) bool {
	switch typeName {
	case "boolean", "byte", "short", "int", "long", "float", "double", "char":
		return false
	default:
		return true
	}
}
