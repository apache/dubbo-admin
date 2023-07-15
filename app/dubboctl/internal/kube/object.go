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
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/apache/dubbo-admin/app/dubboctl/internal/util"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	hashPrompt = "Namespace:Kind:Name=>"
)

// Object wraps k8s Unstructured and exposes the fields we need
type Object struct {
	internal *unstructured.Unstructured

	Namespace string
	Name      string
	Group     string
	Kind      string

	yamlStr string
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

func (obj *Object) Hash() string {
	return strings.Join([]string{obj.Namespace, obj.Kind, obj.Name}, ":")
}

func (obj *Object) YAML() string {
	return obj.yamlStr
}

func (obj *Object) SetNamespace(ns string) {
	obj.Namespace = ns
	obj.internal.SetNamespace(ns)
}

type Objects []*Object

// SortMap generates corresponding map and sorted keys
func (objs Objects) SortMap() (map[string]*Object, []string) {
	var keys []string
	res := make(map[string]*Object)
	for _, obj := range objs {
		if obj.IsValid() {
			hash := obj.Hash()
			res[hash] = obj
			keys = append(keys, hash)
		}
	}
	sort.Strings(keys)
	return res, keys
}

func NewObject(obj *unstructured.Unstructured, yamlStr string) *Object {
	newObj := &Object{
		internal: obj,
		yamlStr:  yamlStr,
	}
	newObj.Namespace = obj.GetNamespace()
	newObj.Name = obj.GetName()

	gvk := obj.GroupVersionKind()
	newObj.Group = gvk.Group
	newObj.Kind = gvk.Kind

	return newObj
}

// ParseObjectsFromManifest parse Objects from manifest which divided by YAML separator "\n---\n"
func ParseObjectsFromManifest(manifest string, continueOnErr bool) (Objects, error) {
	segments, err := util.SplitYAML(manifest)
	if err != nil {
		return nil, err
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
	newObj := NewObject(internal, manifest)
	return newObj, nil
}

// CompareObjects compares object lists and returns diff, add, err using util.DiffYAML.
// It compares objects with same hash value(Namespace:Kind:Name) and returns diff.
// For objects that only one list have, it returns add.
// For objects that could not be parsed successfully, it returns err.
// Refer to TestCompareObjects for examples.
func CompareObjects(objsA, objsB Objects) (string, string, string) {
	var diffRes strings.Builder
	var addRes strings.Builder
	var errRes strings.Builder
	sep := "\n------\n"
	mapA, keysA := objsA.SortMap()
	mapB, keysB := objsB.SortMap()

	for _, hashA := range keysA {
		objA := mapA[hashA]
		if objB, ok := mapB[hashA]; ok {
			diff, err := util.DiffYAML(objA.YAML(), objB.YAML())
			if err != nil {
				errRes.WriteString(fmt.Sprintf("%s%s parse failed, err:\n%s", hashPrompt, hashA, err))
				errRes.WriteString(sep)
				continue
			}
			if diff != "" {
				diffRes.WriteString(fmt.Sprintf("%s%s diff:\n", hashPrompt, hashA))
				diffRes.WriteString(diff)
				diffRes.WriteString(sep)
			}
		} else {
			add, err := util.DiffYAML(objA.YAML(), "")
			if err != nil {
				errRes.WriteString(fmt.Sprintf("%s%s in previous section parse failed, err:\n%s", hashPrompt, hashA, err))
				errRes.WriteString(sep)
				continue
			}
			if add != "" {
				addRes.WriteString(fmt.Sprintf("%s%s in previous addition:\n", hashPrompt, hashA))
				addRes.WriteString(add)
				addRes.WriteString(sep)
			}
		}
	}

	for _, hashB := range keysB {
		objB := mapB[hashB]
		if _, ok := mapA[hashB]; !ok {
			add, err := util.DiffYAML(objB.YAML(), "")
			if err != nil {
				errRes.WriteString(fmt.Sprintf("%s%s in next section parse failed, err:\n%s", hashPrompt, hashB, err))
				errRes.WriteString(sep)
				continue
			}
			if add != "" {
				addRes.WriteString(fmt.Sprintf("%s%s in next addition:\n", hashPrompt, hashB))
				addRes.WriteString(add)
				addRes.WriteString(sep)
			}
		}
	}

	return diffRes.String(), addRes.String(), errRes.String()
}

// CompareObject compares two objects and returns diff.
func CompareObject(objA, objB *Object) (string, error) {
	diff, err := util.DiffYAML(objA.YAML(), objB.YAML())
	if err != nil {
		return "", err
	}
	return diff, nil
}
