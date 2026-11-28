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
	"slices"
	"sort"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

type ResolvedContract struct {
	Operation         *meshproto.MCPOperation
	Parameters        []*meshproto.MCPParameterBinding
	ReturnType        string
	RelatedTypes      []*meshproto.MCPTypeDefinition
	InputSchema       *structpb.Struct
	InputFingerprint  string
	OutputFingerprint string
}

func DefaultParameterBindings(parameterTypes []string) []*meshproto.MCPParameterBinding {
	result := make([]*meshproto.MCPParameterBinding, len(parameterTypes))
	for i := range parameterTypes {
		result[i] = &meshproto.MCPParameterBinding{
			Position: int32(i),
			Alias:    fmt.Sprintf("arg%d", i),
		}
	}
	return result
}

func ResolveOperation(
	mesh string,
	operation *meshproto.MCPOperation,
	bindings []*meshproto.MCPParameterBinding,
	resources []*meshresource.ServiceProviderMetadataResource,
) (*ResolvedContract, error) {
	if operation == nil {
		return nil, fmt.Errorf("operation is required")
	}
	if err := validateOperation(operation); err != nil {
		return nil, err
	}
	if bindings == nil {
		bindings = DefaultParameterBindings(operation.ParameterTypes)
	}
	if err := ValidateParameterBindings(bindings, operation.ParameterTypes); err != nil {
		return nil, err
	}

	var resolved *ResolvedContract
	matchedDefinitions := 0
	for _, resource := range resources {
		if !matchesService(resource, mesh, operation) {
			continue
		}
		matchedDefinitions++
		contract, err := resolveDefinition(operation, bindings, resource.Spec)
		if err != nil {
			return nil, fmt.Errorf("provider definition %s: %w", resource.ResourceKey(), err)
		}
		if resolved == nil {
			resolved = contract
			continue
		}
		if contract.InputFingerprint != resolved.InputFingerprint ||
			contract.OutputFingerprint != resolved.OutputFingerprint {
			return nil, fmt.Errorf("provider definitions disagree for operation %s", operationName(operation))
		}
	}
	if matchedDefinitions == 0 {
		return nil, fmt.Errorf("no provider definition found for operation %s", operationName(operation))
	}
	return resolved, nil
}

func ValidateParameterBindings(bindings []*meshproto.MCPParameterBinding, parameterTypes []string) error {
	if len(bindings) != len(parameterTypes) {
		return fmt.Errorf("parameter binding count %d does not match parameter type count %d", len(bindings), len(parameterTypes))
	}
	positions := make([]bool, len(parameterTypes))
	aliases := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		if binding == nil {
			return fmt.Errorf("parameter binding is nil")
		}
		position := int(binding.Position)
		if position < 0 || position >= len(parameterTypes) {
			return fmt.Errorf("parameter position %d is out of range", position)
		}
		if positions[position] {
			return fmt.Errorf("parameter position %d is duplicated", position)
		}
		positions[position] = true
		alias := strings.TrimSpace(binding.Alias)
		if alias == "" {
			return fmt.Errorf("parameter alias at position %d is empty", position)
		}
		if _, exists := aliases[alias]; exists {
			return fmt.Errorf("parameter alias %q is duplicated", alias)
		}
		aliases[alias] = struct{}{}
	}
	return nil
}

func validateOperation(operation *meshproto.MCPOperation) error {
	if strings.TrimSpace(operation.ServiceName) == "" {
		return fmt.Errorf("service name is required")
	}
	if strings.TrimSpace(operation.MethodName) == "" {
		return fmt.Errorf("method name is required")
	}
	if operation.Protocol != "dubbo" && operation.Protocol != "tri" {
		return fmt.Errorf("unsupported protocol %q", operation.Protocol)
	}
	for i, parameterType := range operation.ParameterTypes {
		if strings.TrimSpace(parameterType) == "" {
			return fmt.Errorf("parameter type at position %d is empty", i)
		}
	}
	return nil
}

func matchesService(
	resource *meshresource.ServiceProviderMetadataResource,
	mesh string,
	operation *meshproto.MCPOperation,
) bool {
	if resource == nil || resource.Spec == nil || resource.Mesh != mesh {
		return false
	}
	return resource.Spec.ServiceName == operation.ServiceName &&
		resource.Spec.Group == operation.Group &&
		resource.Spec.Version == operation.Version
}

func resolveDefinition(
	operation *meshproto.MCPOperation,
	bindings []*meshproto.MCPParameterBinding,
	definition *meshproto.ServiceProviderMetadata,
) (*ResolvedContract, error) {
	method, err := findMethod(definition.Methods, operation.MethodName, operation.ParameterTypes)
	if err != nil {
		return nil, err
	}
	if method == nil {
		return nil, fmt.Errorf("method %s with the selected parameter types is missing", operation.MethodName)
	}
	types, err := buildTypeIndex(definition.Types)
	if err != nil {
		return nil, err
	}
	inputTypes, err := reachableTypes(operation.ParameterTypes, types)
	if err != nil {
		return nil, err
	}
	outputTypes, err := reachableTypes([]string{method.ReturnType}, types)
	if err != nil {
		return nil, err
	}
	allTypes := mergeTypes(inputTypes, outputTypes)

	schema, err := BuildInputSchema(operation.ParameterTypes, bindings, types)
	if err != nil {
		return nil, err
	}
	if method.ReturnType != "void" && method.ReturnType != "java.lang.Void" {
		if _, err := BuildInputSchema(
			[]string{method.ReturnType},
			DefaultParameterBindings([]string{method.ReturnType}),
			types,
		); err != nil {
			return nil, fmt.Errorf("return type: %w", err)
		}
	}
	inputFingerprint, err := fingerprint(struct {
		ServiceName    string
		Group          string
		Version        string
		MethodName     string
		ParameterTypes []string
		Types          []canonicalType
	}{operation.ServiceName, operation.Group, operation.Version, operation.MethodName, operation.ParameterTypes, canonicalTypes(inputTypes)})
	if err != nil {
		return nil, err
	}
	outputFingerprint, err := fingerprint(struct {
		ReturnType string
		Types      []canonicalType
	}{method.ReturnType, canonicalTypes(outputTypes)})
	if err != nil {
		return nil, err
	}

	return &ResolvedContract{
		Operation:         cloneOperation(operation),
		Parameters:        cloneBindings(bindings),
		ReturnType:        method.ReturnType,
		RelatedTypes:      toProtoTypes(allTypes),
		InputSchema:       schema,
		InputFingerprint:  inputFingerprint,
		OutputFingerprint: outputFingerprint,
	}, nil
}

func findMethod(methods []*meshproto.Method, name string, parameterTypes []string) (*meshproto.Method, error) {
	var match *meshproto.Method
	for _, method := range methods {
		if method != nil && method.Name == name && slices.Equal(method.ParameterTypes, parameterTypes) {
			if match != nil && match.ReturnType != method.ReturnType {
				return nil, fmt.Errorf("method %s has conflicting return types", name)
			}
			match = method
		}
	}
	return match, nil
}

func buildTypeIndex(types []*meshproto.Type) (map[string]*meshproto.MCPTypeDefinition, error) {
	result := make(map[string]*meshproto.MCPTypeDefinition, len(types))
	for _, typ := range types {
		if typ == nil || strings.TrimSpace(typ.Type) == "" {
			continue
		}
		candidate := &meshproto.MCPTypeDefinition{
			Type:       typ.Type,
			Items:      slices.Clone(typ.Items),
			Enums:      slices.Clone(typ.Enums),
			Properties: cloneStringMap(typ.Properties),
		}
		if previous, exists := result[typ.Type]; exists && !equalType(previous, candidate) {
			return nil, fmt.Errorf("type %q has conflicting definitions", typ.Type)
		}
		result[typ.Type] = candidate
	}
	return result, nil
}

func reachableTypes(roots []string, types map[string]*meshproto.MCPTypeDefinition) (map[string]*meshproto.MCPTypeDefinition, error) {
	result := make(map[string]*meshproto.MCPTypeDefinition)
	visiting := make(map[string]bool)
	var visit func(string) error
	visit = func(name string) error {
		if name == "" || isPrimitiveOrVoid(name) {
			return nil
		}
		if _, done := result[name]; done {
			return nil
		}
		definition, exists := types[name]
		if !exists {
			return fmt.Errorf("type definition %q is missing", name)
		}
		if visiting[name] {
			return nil
		}
		visiting[name] = true
		result[name] = definition
		for _, item := range definition.Items {
			if err := visit(item); err != nil {
				return err
			}
		}
		for _, propertyType := range definition.Properties {
			if err := visit(propertyType); err != nil {
				return err
			}
		}
		visiting[name] = false
		return nil
	}
	for _, root := range roots {
		if err := visit(root); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func mergeTypes(groups ...map[string]*meshproto.MCPTypeDefinition) map[string]*meshproto.MCPTypeDefinition {
	result := make(map[string]*meshproto.MCPTypeDefinition)
	for _, group := range groups {
		for name, typ := range group {
			result[name] = typ
		}
	}
	return result
}

type canonicalType struct {
	Type       string
	Items      []string
	Enums      []string
	Properties []canonicalProperty
}

type canonicalProperty struct {
	Name string
	Type string
}

func canonicalTypes(types map[string]*meshproto.MCPTypeDefinition) []canonicalType {
	names := make([]string, 0, len(types))
	for name := range types {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]canonicalType, 0, len(names))
	for _, name := range names {
		typ := types[name]
		properties := make([]canonicalProperty, 0, len(typ.Properties))
		for propertyName, propertyType := range typ.Properties {
			properties = append(properties, canonicalProperty{propertyName, propertyType})
		}
		sort.Slice(properties, func(i, j int) bool { return properties[i].Name < properties[j].Name })
		items := slices.Clone(typ.Items)
		enums := slices.Clone(typ.Enums)
		sort.Strings(enums)
		result = append(result, canonicalType{typ.Type, items, enums, properties})
	}
	return result
}

func toProtoTypes(types map[string]*meshproto.MCPTypeDefinition) []*meshproto.MCPTypeDefinition {
	names := make([]string, 0, len(types))
	for name := range types {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]*meshproto.MCPTypeDefinition, 0, len(names))
	for _, name := range names {
		result = append(result, cloneType(types[name]))
	}
	return result
}

func fingerprint(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func operationName(operation *meshproto.MCPOperation) string {
	return fmt.Sprintf("%s:%s:%s.%s(%s)", operation.ServiceName, operation.Version, operation.Group,
		operation.MethodName, strings.Join(operation.ParameterTypes, ","))
}

func equalType(left, right *meshproto.MCPTypeDefinition) bool {
	return left.Type == right.Type && slices.Equal(left.Items, right.Items) &&
		slices.Equal(left.Enums, right.Enums) && mapsEqual(left.Properties, right.Properties)
}

func mapsEqual(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func cloneOperation(operation *meshproto.MCPOperation) *meshproto.MCPOperation {
	return &meshproto.MCPOperation{
		ServiceName:    operation.ServiceName,
		Group:          operation.Group,
		Version:        operation.Version,
		MethodName:     operation.MethodName,
		ParameterTypes: slices.Clone(operation.ParameterTypes),
		Protocol:       operation.Protocol,
	}
}

func cloneBindings(bindings []*meshproto.MCPParameterBinding) []*meshproto.MCPParameterBinding {
	result := make([]*meshproto.MCPParameterBinding, len(bindings))
	for i, binding := range bindings {
		if binding != nil {
			result[i] = &meshproto.MCPParameterBinding{
				Position:    binding.Position,
				Alias:       binding.Alias,
				Description: binding.Description,
			}
		}
	}
	return result
}

func cloneType(typ *meshproto.MCPTypeDefinition) *meshproto.MCPTypeDefinition {
	return &meshproto.MCPTypeDefinition{
		Type:       typ.Type,
		Items:      slices.Clone(typ.Items),
		Enums:      slices.Clone(typ.Enums),
		Properties: cloneStringMap(typ.Properties),
	}
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return nil
	}
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}
