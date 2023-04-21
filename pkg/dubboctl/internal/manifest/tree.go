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

// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/apache/dubbo-admin/pkg/dubboctl/internal/util"

	"gopkg.in/yaml.v2"
	yaml2 "sigs.k8s.io/yaml"
)

// PathContext provides a means for traversing a tree towards the root.
type PathContext struct {
	// Parent in the Parent of this PathContext.
	Parent *PathContext
	// KeyToChild is the key required to reach the child.
	KeyToChild interface{}
	// Node is the actual Node in the data tree.
	Node interface{}
}

// String implements the Stringer interface.
func (nc *PathContext) String() string {
	ret := "\n--------------- NodeContext ------------------\n"
	if nc.Parent != nil {
		ret += fmt.Sprintf("Parent.Node=\n%s\n", nc.Parent.Node)
		ret += fmt.Sprintf("KeyToChild=%v\n", nc.Parent.KeyToChild)
	}

	ret += fmt.Sprintf("Node=\n%s\n", nc.Node)
	ret += "----------------------------------------------\n"

	return ret
}

// GetPathContext returns the PathContext for the Node which has the given path from root.
// It returns false and no error if the given path is not found, or an error code in other error situations, like
// a malformed path.
// It also creates a tree of PathContexts during the traversal so that Parent nodes can be updated if required. This is
// required when (say) appending to a list, where the parent list itself must be updated.
func GetPathContext(root interface{}, path util.Path, createMissing bool) (*PathContext, bool, error) {
	return getPathContext(&PathContext{Node: root}, path, path, createMissing)
}

// WritePathContext writes the given value to the Node in the given PathContext.
func WritePathContext(nc *PathContext, value interface{}, merge bool) error {
	if !util.IsValueNil(value) {
		return setPathContext(nc, value, merge)
	}

	if nc.Parent == nil {
		return errors.New("cannot delete root element")
	}

	switch {
	case isSliceOrPtrInterface(nc.Parent.Node):
		if err := util.DeleteFromSlicePtr(nc.Parent.Node, nc.Parent.KeyToChild.(int)); err != nil {
			return err
		}
		if isMapOrInterface(nc.Parent.Parent.Node) {
			return util.InsertIntoMap(nc.Parent.Parent.Node, nc.Parent.Parent.KeyToChild, nc.Parent.Node)
		}
		// TODO: The case of deleting a list.list.node element is not currently supported.
		return fmt.Errorf("cannot delete path: unsupported parent.parent type %T for delete", nc.Parent.Parent.Node)
	case util.IsMap(nc.Parent.Node):
		return util.DeleteFromMap(nc.Parent.Node, nc.Parent.KeyToChild)
	default:
	}
	return fmt.Errorf("cannot delete path: unsupported parent type %T for delete", nc.Parent.Node)
}

// WriteNode writes value to the tree in root at the given path, creating any required missing internal nodes in path.
func WriteNode(root interface{}, path util.Path, value interface{}) error {
	pc, _, err := getPathContext(&PathContext{Node: root}, path, path, true)
	if err != nil {
		return err
	}
	return WritePathContext(pc, value, false)
}

// MergeNode merges value to the tree in root at the given path, creating any required missing internal nodes in path.
func MergeNode(root interface{}, path util.Path, value interface{}) error {
	pc, _, err := getPathContext(&PathContext{Node: root}, path, path, true)
	if err != nil {
		return err
	}
	return WritePathContext(pc, value, true)
}

// Find returns the value at path from the given tree, or false if the path does not exist.
// It behaves differently from GetPathContext in that it never creates map entries at the leaf and does not provide
// a way to mutate the parent of the found node.
func Find(inputTree map[string]interface{}, path util.Path) (interface{}, bool, error) {
	if len(path) == 0 {
		return nil, false, fmt.Errorf("path is empty")
	}
	node, found := find(inputTree, path)
	return node, found, nil
}

// Delete sets value at path of input untyped tree to nil
func Delete(root map[string]interface{}, path util.Path) (bool, error) {
	pc, _, err := getPathContext(&PathContext{Node: root}, path, path, false)
	if err != nil {
		return false, err
	}
	return true, WritePathContext(pc, nil, false)
}

// getPathContext is the internal implementation of GetPathContext.
// If createMissing is true, it creates any missing map (but NOT list) path entries in root.
func getPathContext(nc *PathContext, fullPath, remainPath util.Path, createMissing bool) (*PathContext, bool, error) {
	if len(remainPath) == 0 {
		return nc, true, nil
	}
	pe := remainPath[0]

	if nc.Node == nil {
		if !createMissing {
			return nil, false, fmt.Errorf("node %s is zero", pe)
		}
		if util.IsNPathElement(pe) || util.IsKVPathElement(pe) {
			nc.Node = []interface{}{}
		} else {
			nc.Node = make(map[string]interface{})
		}
	}

	v := reflect.ValueOf(nc.Node)
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	ncNode := v.Interface()

	// For list types, we need a key to identify the selected list item. This can be either a value key of the
	// form :matching_value in the case of a leaf list, or a matching key:value in the case of a non-leaf list.
	if lst, ok := ncNode.([]interface{}); ok {
		// If the path element has the form [N], a list element is being selected by index. Return the element at index
		// N if it exists.
		if util.IsNPathElement(pe) {
			idx, err := util.PathN(pe)
			if err != nil {
				return nil, false, fmt.Errorf("path %s, index %s: %s", fullPath, pe, err)
			}
			var foundNode interface{}
			if idx >= len(lst) || idx < 0 {
				if !createMissing {
					return nil, false, fmt.Errorf("index %d exceeds list length %d at path %s", idx, len(lst), remainPath)
				}
				idx = len(lst)
				foundNode = make(map[string]interface{})
			} else {
				foundNode = lst[idx]
			}
			nn := &PathContext{
				Parent: nc,
				Node:   foundNode,
			}
			nc.KeyToChild = idx
			return getPathContext(nn, fullPath, remainPath[1:], createMissing)
		}

		// Otherwise the path element must have form [key:value]. In this case, go through all list elements, which
		// must have map type, and try to find one which has a matching key:value.
		for idx, le := range lst {
			// non-leaf list, expect to match item by key:value.
			if lm, ok := le.(map[interface{}]interface{}); ok {
				k, v, err := util.PathKV(pe)
				if err != nil {
					return nil, false, fmt.Errorf("path %s: %s", fullPath, err)
				}
				if stringsEqual(lm[k], v) {
					nn := &PathContext{
						Parent: nc,
						Node:   lm,
					}
					nc.KeyToChild = idx
					nn.KeyToChild = k
					if len(remainPath) == 1 {
						return nn, true, nil
					}
					return getPathContext(nn, fullPath, remainPath[1:], createMissing)
				}
				continue
			}
			// repeat of the block above for the case where tree unmarshals to map[string]interface{}. There doesn't
			// seem to be a way to merge this case into the above block.
			if lm, ok := le.(map[string]interface{}); ok {
				k, v, err := util.PathKV(pe)
				if err != nil {
					return nil, false, fmt.Errorf("path %s: %s", fullPath, err)
				}
				if stringsEqual(lm[k], v) {
					nn := &PathContext{
						Parent: nc,
						Node:   lm,
					}
					nc.KeyToChild = idx
					nn.KeyToChild = k
					if len(remainPath) == 1 {
						return nn, true, nil
					}
					return getPathContext(nn, fullPath, remainPath[1:], createMissing)
				}
				continue
			}
			// leaf list, expect path element [V], match based on value V.
			v, err := util.PathV(pe)
			if err != nil {
				return nil, false, fmt.Errorf("path %s: %s", fullPath, err)
			}
			if matchesRegex(v, le) {
				nn := &PathContext{
					Parent: nc,
					Node:   le,
				}
				nc.KeyToChild = idx
				return getPathContext(nn, fullPath, remainPath[1:], createMissing)
			}
		}
		return nil, false, fmt.Errorf("path %s: element %s not found", fullPath, pe)
	}

	if util.IsMap(ncNode) {
		var nn interface{}
		if m, ok := ncNode.(map[interface{}]interface{}); ok {
			nn, ok = m[pe]
			if !ok {
				// remainPath == 1 means the patch is creation of a new leaf.
				if createMissing || len(remainPath) == 1 {
					m[pe] = make(map[interface{}]interface{})
					nn = m[pe]
				} else {
					return nil, false, fmt.Errorf("path not found at element %s in path %s", pe, fullPath)
				}
			}
		}
		if reflect.ValueOf(ncNode).IsNil() {
			ncNode = make(map[string]interface{})
			nc.Node = ncNode
		}
		if m, ok := ncNode.(map[string]interface{}); ok {
			nn, ok = m[pe]
			if !ok {
				// remainPath == 1 means the patch is creation of a new leaf.
				if createMissing || len(remainPath) == 1 {
					nextElementNPath := len(remainPath) > 1 && util.IsNPathElement(remainPath[1])
					if nextElementNPath {
						m[pe] = make([]interface{}, 0)
					} else {
						m[pe] = make(map[string]interface{})
					}
					nn = m[pe]
				} else {
					return nil, false, fmt.Errorf("path not found at element %s in path %s", pe, fullPath)
				}
			}
		}

		npc := &PathContext{
			Parent: nc,
			Node:   nn,
		}
		// for slices, use the address so that the slice can be mutated.
		if util.IsSlice(nn) {
			npc.Node = &nn
		}
		nc.KeyToChild = pe
		return getPathContext(npc, fullPath, remainPath[1:], createMissing)
	}

	return nil, false, fmt.Errorf("leaf type %T in non-leaf Node %s", nc.Node, remainPath)
}

// setPathContext writes the given value to the Node in the given PathContext,
// enlarging all PathContext lists to ensure all indexes are valid.
func setPathContext(nc *PathContext, value interface{}, merge bool) error {
	processParent, err := setValueContext(nc, value, merge)
	if err != nil || !processParent {
		return err
	}

	// If the path included insertions, process them now
	if nc.Parent.Parent == nil {
		return nil
	}
	return setPathContext(nc.Parent, nc.Parent.Node, false) // note: tail recursive
}

// setValueContext writes the given value to the Node in the given PathContext.
// If setting the value requires growing the final slice, grows it.
func setValueContext(nc *PathContext, value interface{}, merge bool) (bool, error) {
	if nc.Parent == nil {
		return false, nil
	}

	vv, mapFromString := tryToUnmarshalStringToYAML(value)

	switch parentNode := nc.Parent.Node.(type) {
	case *interface{}:
		switch vParentNode := (*parentNode).(type) {
		case []interface{}:
			idx := nc.Parent.KeyToChild.(int)
			if idx == -1 {
				// Treat -1 as insert-at-end of list
				idx = len(vParentNode)
			}

			if idx >= len(vParentNode) {
				newElements := make([]interface{}, idx-len(vParentNode)+1)
				vParentNode = append(vParentNode, newElements...)
				*parentNode = vParentNode
			}

			merged, err := mergeConditional(vv, nc.Node, merge)
			if err != nil {
				return false, err
			}

			vParentNode[idx] = merged
			nc.Node = merged
		default:
			return false, fmt.Errorf("don't know about vtype %T", vParentNode)
		}
	case map[string]interface{}:
		key := nc.Parent.KeyToChild.(string)

		// Update is treated differently depending on whether the value is a scalar or map type. If scalar,
		// insert a new element into the terminal node, otherwise replace the terminal node with the new subtree.
		if ncNode, ok := nc.Node.(*interface{}); ok && !mapFromString {
			switch vNcNode := (*ncNode).(type) {
			case []interface{}:
				switch vv.(type) {
				case map[string]interface{}:
					// the vv is a map, and the node is a slice
					mergedValue := append(vNcNode, vv)
					parentNode[key] = mergedValue
				case *interface{}:
					merged, err := mergeConditional(vv, vNcNode, merge)
					if err != nil {
						return false, err
					}

					parentNode[key] = merged
					nc.Node = merged
				default:
					// the vv is an basic JSON type (int, float, string, bool)
					vv = append(vNcNode, vv)
					parentNode[key] = vv
					nc.Node = vv
				}
			default:
				return false, fmt.Errorf("don't know about vnc type %T", vNcNode)
			}
		} else {
			// For map passed as string type, the root is the new key.
			if mapFromString {
				if err := util.DeleteFromMap(nc.Parent.Node, nc.Parent.KeyToChild); err != nil {
					return false, err
				}
				vm := vv.(map[string]interface{})
				newKey := getTreeRoot(vm)
				return false, util.InsertIntoMap(nc.Parent.Node, newKey, vm[newKey])
			}
			parentNode[key] = vv
			nc.Node = vv
		}
	// TODO `map[interface{}]interface{}` is used by tests in operator/cmd/mesh, we should add our own tests
	case map[interface{}]interface{}:
		key := nc.Parent.KeyToChild.(string)
		parentNode[key] = vv
		nc.Node = vv
	default:
		return false, fmt.Errorf("don't know about type %T", parentNode)
	}

	return true, nil
}

// mergeConditional returns a merge of newVal and originalVal if merge is true, otherwise it returns newVal.
func mergeConditional(newVal, originalVal interface{}, merge bool) (interface{}, error) {
	if !merge || util.IsValueNilOrDefault(originalVal) {
		return newVal, nil
	}
	newS, err := yaml.Marshal(newVal)
	if err != nil {
		return nil, err
	}
	if util.IsYAMLEmpty(string(newS)) {
		return originalVal, nil
	}
	originalS, err := yaml.Marshal(originalVal)
	if err != nil {
		return nil, err
	}
	if util.IsYAMLEmpty(string(originalS)) {
		return newVal, nil
	}

	mergedS, err := util.OverlayYAML(string(originalS), string(newS))
	if err != nil {
		return nil, err
	}

	if util.IsMap(originalVal) {
		// For JSON compatibility
		out := make(map[string]interface{})
		if err := yaml.Unmarshal([]byte(mergedS), &out); err != nil {
			return nil, err
		}
		return out, nil
	}
	// For scalars and slices, copy the type
	out := originalVal
	if err := yaml.Unmarshal([]byte(mergedS), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// find returns the value at path from the given tree, or false if the path does not exist.
func find(treeNode interface{}, path util.Path) (interface{}, bool) {
	if len(path) == 0 || treeNode == nil {
		return nil, false
	}
	switch nt := treeNode.(type) {
	case map[interface{}]interface{}:
		val := nt[path[0]]
		if val == nil {
			return nil, false
		}
		if len(path) == 1 {
			return val, true
		}
		return find(val, path[1:])
	case map[string]interface{}:
		val := nt[path[0]]
		if val == nil {
			return nil, false
		}
		if len(path) == 1 {
			return val, true
		}
		return find(val, path[1:])
	case []interface{}:
		idx, err := strconv.Atoi(path[0])
		if err != nil {
			return nil, false
		}
		if idx >= len(nt) {
			return nil, false
		}
		val := nt[idx]
		return find(val, path[1:])
	default:
		return nil, false
	}
}

// stringsEqual reports whether the string representations of a and b are equal. a and b may have different types.
func stringsEqual(a, b interface{}) bool {
	return fmt.Sprint(a) == fmt.Sprint(b)
}

// matchesRegex reports whether str regex matches pattern.
func matchesRegex(pattern, str interface{}) bool {
	match, err := regexp.MatchString(fmt.Sprint(pattern), fmt.Sprint(str))
	if err != nil {
		// log.Errorf("bad regex expression %s", fmt.Sprint(pattern))
		return false
	}
	return match
}

// isSliceOrPtrInterface reports whether v is a slice, a ptr to slice or interface to slice.
func isSliceOrPtrInterface(v interface{}) bool {
	vv := reflect.ValueOf(v)
	if vv.Kind() == reflect.Ptr {
		vv = vv.Elem()
	}
	if vv.Kind() == reflect.Interface {
		vv = vv.Elem()
	}
	return vv.Kind() == reflect.Slice
}

// isMapOrInterface reports whether v is a map, or interface to a map.
func isMapOrInterface(v interface{}) bool {
	vv := reflect.ValueOf(v)
	if vv.Kind() == reflect.Interface {
		vv = vv.Elem()
	}
	return vv.Kind() == reflect.Map
}

// tryToUnmarshalStringToYAML tries to unmarshal something that may be a YAML list or map into a structure. If not
// possible, returns original scalar value.
func tryToUnmarshalStringToYAML(s interface{}) (interface{}, bool) {
	// If value type is a string it could either be a literal string or a map type passed as a string. Try to unmarshal
	// to discover it's the latter.
	vv := s

	if reflect.TypeOf(vv).Kind() == reflect.String {
		sv := strings.Split(vv.(string), "\n")
		// Need to be careful not to transform string literals into maps unless they really are maps, since scalar handling
		// is different for inserts.
		if len(sv) == 1 && strings.Contains(s.(string), ": ") ||
			len(sv) > 1 && strings.Contains(s.(string), ":") {
			nv := make(map[string]interface{})
			if err := json.Unmarshal([]byte(vv.(string)), &nv); err == nil {
				// treat JSON as string
				return vv, false
			}
			if err := yaml2.Unmarshal([]byte(vv.(string)), &nv); err == nil {
				return nv, true
			}
		}
	}
	// looks like a literal or failed unmarshal, return original type.
	return vv, false
}

// getTreeRoot returns the first key found in m. It assumes a single root tree.
func getTreeRoot(m map[string]interface{}) string {
	for k := range m {
		return k
	}
	return ""
}
