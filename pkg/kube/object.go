// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kube

import (
	"bufio"
	"bytes"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"strings"
)

type Objects []*Object

type Object struct {
	internal *unstructured.Unstructured

	Namespace string
	Name      string
	Group     string
	Kind      string
}

func (obj *Object) IsValid() bool {
	return obj.Kind != ""
}

func (obj *Object) IsEqual(another *Object) bool {
	if obj == nil {
		return another == nil
	}
	if another == nil {
		return false
	}
	if obj.Namespace != another.Namespace ||
		obj.Name != another.Name || obj.Group != another.Group || obj.Kind != another.Kind {
		return false
	}
	if obj.Unstructured() == nil {
		return another.Unstructured() == nil
	}
	if another.Unstructured() == nil {
		return false
	}
	objJson, err := obj.Unstructured().MarshalJSON()
	if err != nil {
		return false
	}
	anotherJson, err := another.Unstructured().MarshalJSON()
	if err != nil {
		return false
	}
	return bytes.Equal(objJson, anotherJson)
}

func (obj *Object) Unstructured() *unstructured.Unstructured {
	return obj.internal
}

func NewObject(obj *unstructured.Unstructured) *Object {
	newObj := &Object{internal: obj}
	newObj.Namespace = obj.GetNamespace()
	newObj.Name = obj.GetName()

	gvk := obj.GroupVersionKind()
	newObj.Group = gvk.Group
	newObj.Kind = gvk.Kind

	return newObj
}

// ParseObjectsFromManifest parse Objects from manifest which divided by YAML separator "\n---\n"
func ParseObjectsFromManifest(manifest string, continueOnErr bool) (Objects, error) {
	var buf bytes.Buffer
	var segments []string
	scanner := bufio.NewScanner(strings.NewReader(manifest))
	for scanner.Scan() {
		line := scanner.Text()
		// ignore comment line and empty line
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		// yaml separator
		if strings.HasPrefix(line, "---") {
			segments = append(segments, buf.String())
			buf.Reset()
			continue
		}
		if _, err := buf.WriteString(line + "\n"); err != nil {
			return nil, err
		}
	}
	// add the last yaml(not empty)
	if buf.String() != "" {
		segments = append(segments, buf.String())
	}

	var objects Objects
	for _, segment := range segments {
		newObj, err := ParseObjectFromManifest(segment)
		if err != nil {
			if !continueOnErr {
				return nil, err
			}
			continue
		}
		if newObj.IsValid() {
			objects = append(objects, newObj)
		}
	}

	return objects, nil
}

// ParseObjectFromManifest parse Object from manifest which represents a single K8s object
func ParseObjectFromManifest(manifest string) (*Object, error) {
	reader := strings.NewReader(manifest)
	decoder := yaml.NewYAMLOrJSONDecoder(reader, 1024)

	internal := &unstructured.Unstructured{}
	if err := decoder.Decode(internal); err != nil {
		return nil, err
	}
	newObj := NewObject(internal)
	return newObj, nil
}
