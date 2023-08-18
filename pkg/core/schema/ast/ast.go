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

package ast

import (
	"encoding/json"

	"github.com/apache/dubbo-admin/pkg/core/tools/resource"
	"github.com/apache/dubbo-admin/pkg/core/validation"
	"github.com/ghodss/yaml"
)

// Metadata is the top-level container
type Metadata struct {
	Collections []*Collection `json:"collections"`
	Resources   []*Resource   `json:"resources"`
}

// for testing purposes
var jsonUnmarshal = json.Unmarshal

// FindResourceForGroupKind looks up a resource with the given group and kind. Returns nil if not found.
func (m *Metadata) FindResourceForGroupKind(group, kind string) *Resource {
	for _, r := range m.Resources {
		if r.Group == group && r.Kind == kind {
			return r
		}
	}
	return nil
}

func (m *Metadata) UnmarshalJSON(data []byte) error {
	var in struct {
		Collections []*Collection `json:"collections"`
		Resources   []*Resource   `json:"resources"`
	}

	if err := jsonUnmarshal(data, &in); err != nil {
		return err
	}

	m.Collections = in.Collections
	m.Resources = in.Resources
	// Process resources.
	for i, r := range m.Resources {
		if r.Validate == "" {
			validateFn := "Validate" + asResourceVariableName(r.Kind)
			if !validation.IsValidateFunc(validateFn) {
				validateFn = "EmptyValidate"
			}
			m.Resources[i].Validate = validateFn
		}
	}

	// Process collections
	for i, c := range m.Collections {
		// If no variable name was specified, use default.
		if c.VariableName == "" {
			m.Collections[i].VariableName = asCollectionVariableName(c.Name)
		}
	}
	return nil
}

var _ json.Unmarshaler = &Metadata{}

// Collection metadata. Describes basic structure of collections.
type Collection struct {
	Name         string `json:"name"`
	VariableName string `json:"variableName"`
	Group        string `json:"group"`
	Kind         string `json:"kind"`
	Dds          bool   `json:"dds"`
}

// Resource metadata for resources contained within a collection.
type Resource struct {
	Group         string `json:"group"`
	Version       string `json:"version"`
	Kind          string `json:"kind"`
	Plural        string `json:"plural"`
	ClusterScoped bool   `json:"clusterScoped"`
	Proto         string `json:"proto"`
	Validate      string `json:"validate"`
}

func asResourceVariableName(n string) string {
	return resource.CamelCase(n)
}

func asCollectionVariableName(n string) string {
	n = resource.CamelCaseWithSeparator(n, "/")
	n = resource.CamelCaseWithSeparator(n, ".")
	return n
}

// Parse and return a yaml representation of Metadata
func Parse(yamlText string) (*Metadata, error) {
	var s Metadata
	err := yaml.Unmarshal([]byte(yamlText), &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
