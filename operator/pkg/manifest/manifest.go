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

package manifest

import (
	"encoding/json"

	"github.com/apache/dubbo-admin/operator/pkg/component"
	"github.com/apache/dubbo-admin/operator/pkg/util"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

type Manifest struct {
	*unstructured.Unstructured
	Content string
}

type ManifestSet struct {
	Components component.Name
	Manifests  []Manifest
}

func FromObject(us *unstructured.Unstructured) (Manifest, error) {
	c, err := yaml.Marshal(us)
	if err != nil {
		return Manifest{}, err
	}
	return Manifest{
		Unstructured: us,
		Content:      string(c),
	}, nil
}

// ParseMultiple splits a string containing potentially many YAML objects, and parses them.
func ParseMultiple(output string) ([]Manifest, error) {
	return Parse(util.SplitString(output))
}

// Parse parses a list of YAML objects.
func Parse(output []string) ([]Manifest, error) {
	result := make([]Manifest, 0, len(output))
	for _, m := range output {
		mf, err := FromYAML([]byte(m))
		if err != nil {
			return nil, err
		}
		if mf.GetObjectKind().GroupVersionKind().Kind == "" {
			// This is not an object. Could be empty template, comments only, etc
			continue
		}
		result = append(result, mf)
	}
	return result, nil
}

func (m Manifest) Hash() string {
	return ObjectHash(m.Unstructured)
}

func ObjectHash(o *unstructured.Unstructured) string {
	k := o.GroupVersionKind().Kind
	switch o.GroupVersionKind().Kind {
	case "ClusterRole", "ClusterRoleBinding":
		return k + ":" + o.GetName()
	}
	return k + ":" + o.GetNamespace() + ":" + o.GetName()
}

func FromJSON(j []byte) (Manifest, error) {
	us := &unstructured.Unstructured{}
	if err := json.Unmarshal(j, us); err != nil {
		return Manifest{}, err
	}
	y, err := yaml.Marshal(us)
	if err != nil {
		return Manifest{}, err
	}
	return Manifest{Unstructured: us, Content: string(y)}, nil
}

func FromYAML(y []byte) (Manifest, error) {
	us := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(y, us); err != nil {
		return Manifest{}, err
	}
	return Manifest{
		Unstructured: us,
		Content:      string(y),
	}, nil
}
