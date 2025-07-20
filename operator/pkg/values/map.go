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

package values

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/apache/dubbo-admin/operator/pkg/util/ptr"
	"sigs.k8s.io/yaml"
)

// Map is a wrapper around an untyped map, used throughout the operator codebase for generic access.
type Map map[string]any

// JSON serializes a Map to a JSON string.
func (m Map) JSON() string {
	bytes, err := json.Marshal(m)
	if err != nil {
		panic(fmt.Sprintf("json Marshal: %v", err))
	}
	return string(bytes)
}

// YAML serializes a Map to a YAML string.
func (m Map) YAML() string {
	bytes, err := yaml.Marshal(m)
	if err != nil {
		panic(fmt.Sprintf("yaml Marshal: %v", err))
	}
	return string(bytes)
}

// MapFromJSON constructs a Map from JSON
func MapFromJSON(input []byte) (Map, error) {
	m := make(Map)
	err := json.Unmarshal(input, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// MapFromYAML constructs a Map from YAML
func MapFromYAML(input []byte) (Map, error) {
	m := make(Map)
	err := yaml.Unmarshal(input, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func tableLookup(m Map, simple string) (Map, bool) {
	v, ok := m[simple]
	if !ok {
		return nil, false
	}
	if vv, ok := v.(map[string]interface{}); ok {
		return vv, true
	}
	// This catches a case where a value is of type Values, but doesn't (for some
	// reason) match the map[string]interface{}. This has been observed in the
	// wild, and might be a result of a nil map of type Values.
	if vv, ok := v.(Map); ok {
		return vv, true
	}
	return nil, false
}

func parsePath(key string) []string { return strings.Split(key, ".") }

func (m Map) GetPathMap(s string) (Map, bool) {
	current := m
	for _, n := range parsePath(s) {
		subkey, ok := tableLookup(current, n)
		if !ok {
			return nil, false
		}
		current = subkey
	}
	return current, true
}

func splitEscaped(s string, r rune) []string {
	var prev rune
	if len(s) == 0 {
		return []string{}
	}
	prevIndex := 0
	var out []string
	for i, c := range s {
		if c == r && (i == 0 || i > 0 && prev != '\\') {
			out = append(out, s[prevIndex:i])
			prevIndex = i + 1
		}
		prev = c
	}
	out = append(out, s[prevIndex:])
	return out
}

func splitPath(path string) []string {
	path = filepath.Clean(path)
	path = strings.TrimPrefix(path, ".")
	path = strings.TrimSuffix(path, ".")
	pv := splitEscaped(path, '.')
	var r []string
	for _, str := range pv {
		if str != "" {
			str = strings.ReplaceAll(str, "\\.", ".")
			// Is str of the form node[expr], convert to "node", "[expr]"?
			nBracket := strings.IndexRune(str, '[')
			if nBracket > 0 {
				r = append(r, str[:nBracket], str[nBracket:])
			} else {
				// str is "[expr]" or "node"
				r = append(r, str)
			}
		}
	}
	return r
}

// GetPathAs is a helper function to get a patch value and cast it to a specified type.
// If the path is not found, or the cast fails, the zero value is returned.
func GetPathAs[T any](m Map, name string) T {
	v, ok := m.GetPath(name)
	if !ok {
		return ptr.Empty[T]()
	}
	t, _ := v.(T)
	return t
}

// GetPathString is a helper around TryGetPathAs[string] to allow usage as a method (otherwise impossible with generics)
func (m Map) GetPathString(s string) string {
	return GetPathAs[string](m, s)
}

// GetPathStringOr is a helper around TryGetPathAs[string] to allow usage as a method (otherwise impossible with generics),
// with an allowance for a default value if it is not found/not set.
func (m Map) GetPathStringOr(s string, def string) string {
	return ptr.NonEmptyOrDefault(m.GetPathString(s), def)
}

func (m Map) GetPath(name string) (any, bool) {
	current := any(m)
	paths := splitPath(name)
	for _, n := range paths {
		if idx, ok := extractIndex(n); ok {
			a, ok := current.([]any)
			if !ok {
				return nil, false
			}
			if idx >= 0 && idx < len(a) {
				current = a[idx]
			} else {
				return nil, false
			}
		} else if k, v, ok := extractKeyValue(n); ok {
			a, ok := current.([]any)
			if !ok {
				return nil, false
			}
			index := -1
			for idx, cm := range a {
				if MustCastAsMap(cm)[k] == v {
					index = idx
					break
				}
			}
			if index == -1 {
				return nil, false
			}
			current = a[idx]
		} else {
			cm, ok := CastAsMap(current)
			if !ok {
				return nil, false
			}
			subKey, ok := cm[n]
			if !ok {
				return nil, false
			}
			current = subKey
		}
	}
	if p, ok := current.(*any); ok {
		return *p, true
	}
	return current, true
}

// MustCastAsMap casts a value to a Map; if the value is not a map, it will panic..
func MustCastAsMap(current any) Map {
	m, ok := CastAsMap(current)
	if !ok {
		if !reflect.ValueOf(current).IsValid() {
			return Map{}
		}
		panic(fmt.Sprintf("not a map, got %T: %v %v", current, current, reflect.ValueOf(current).Kind()))
	}
	return m
}

// CastAsMap casts a value to a Map, if possible.
func CastAsMap(current any) (Map, bool) {
	if m, ok := current.(Map); ok {
		return m, true
	}
	if m, ok := current.(map[string]any); ok {
		return m, true
	}
	return nil, false
}

// ConvertMap translates a Map to a T, via JSON
func ConvertMap[T any](m Map) (T, error) {
	return fromJSON[T]([]byte(m.JSON()))
}

func fromJSON[T any](overlay []byte) (T, error) {
	v := new(T)
	err := json.Unmarshal(overlay, &v)
	if err != nil {
		return ptr.Empty[T](), err
	}
	return *v, nil
}

// getPV returns the path and value components for the given set flag string, which must be in path=value format.
func getPV(setFlag string) (path string, value string) {
	pv := strings.Split(setFlag, "=")
	if len(pv) != 2 {
		return setFlag, ""
	}
	path, value = strings.TrimSpace(pv[0]), strings.TrimSpace(pv[1])
	return
}

// SetPath applies values from a path like `key.subkey`, `key.[0].var`, or `key.[name:foo]`.
func (m Map) SetPath(paths string, value any) error {
	path := splitPath(paths)
	base := m
	if err := setPathRecurse(base, path, value); err != nil {
		return err
	}
	return nil
}

// SetPaths applies values from input like `key.subkey=val`
func (m Map) SetPaths(paths ...string) error {
	for _, sf := range paths {
		p, v := getPV(sf)
		// input value type is always string, transform it to correct type before setting.
		var val any = v
		if !isAlwaysString(p) {
			val = parseValue(v)
		}
		if err := m.SetPath(p, val); err != nil {
			return err
		}
	}
	return nil
}

// SetSpecPaths applies values from input like `key.subkey=val`, and applies them under 'spec'
func (m Map) SetSpecPaths(paths ...string) error {
	for _, path := range paths {
		if err := m.SetPaths("spec." + path); err != nil {
			return err
		}
	}
	return nil
}

func setPathRecurse(base map[string]any, paths []string, value any) error {
	seg := paths[0]
	last := len(paths) == 1
	nextIsArray := len(paths) >= 2 && strings.HasPrefix(paths[1], "[")
	if nextIsArray {
		last = len(paths) == 2
		// Find or create target list
		if _, f := base[seg]; !f {
			base[seg] = []any{}
		}
		var index int
		if k, v, ok := extractKV(paths[1]); ok {
			index = -1
			for idx, cm := range base[seg].([]any) {
				if MustCastAsMap(cm)[k] == v {
					index = idx
					break
				}
			}
			if index == -1 {
				return fmt.Errorf("element %v not found", paths[1])
			}
		} else if idx, ok := extractIndex(paths[1]); ok {
			index = idx
		} else {
			return fmt.Errorf("unknown segment %v", paths[1])
		}
		l := base[seg].([]any)
		if index < 0 || index >= len(l) {
			// Index is greater, we need to append
			if last {
				l = append(l, value)
			} else {
				nm := Map{}
				if err := setPathRecurse(nm, paths[2:], value); err != nil {
					return err
				}
				l = append(l, nm)
			}
			base[seg] = l
		} else {
			v := MustCastAsMap(l[index])
			if err := setPathRecurse(v, paths[2:], value); err != nil {
				return err
			}
			l[index] = v
		}
	} else {
		if _, f := base[seg]; !f {
			base[seg] = map[string]any{}
		}
		if last {
			base[seg] = value
		} else {
			return setPathRecurse(MustCastAsMap(base[seg]), paths[1:], value)
		}
	}
	return nil
}

func extractKV(seg string) (string, string, bool) {
	if !strings.HasPrefix(seg, "[") || !strings.HasSuffix(seg, "]") {
		return "", "", false
	}
	sanitized := seg[1 : len(seg)-1]
	return strings.Cut(sanitized, ":")
}

func extractIndex(seg string) (int, bool) {
	if !strings.HasPrefix(seg, "[") || !strings.HasSuffix(seg, "]") {
		return 0, false
	}
	sanitized := seg[1 : len(seg)-1]
	v, err := strconv.Atoi(sanitized)
	if err != nil {
		return 0, false
	}
	return v, true
}

func extractKeyValue(seg string) (string, string, bool) {
	if !strings.HasPrefix(seg, "[") || !strings.HasSuffix(seg, "]") {
		return "", "", false
	}
	sanitized := seg[1 : len(seg)-1]
	return strings.Cut(sanitized, ":")
}

// alwaysString represents types that should always be decoded as strings
var alwaysString = []string{}

func isAlwaysString(s string) bool {
	for _, a := range alwaysString {
		if strings.HasPrefix(s, a) {
			return true
		}
	}
	return false
}

// parseValue parses string into a value.
func parseValue(valueStr string) any {
	var value any
	if v, err := strconv.Atoi(valueStr); err == nil {
		value = v
	} else if v, err := strconv.ParseFloat(valueStr, 64); err == nil {
		value = v
	} else if v, err := strconv.ParseBool(valueStr); err == nil {
		value = v
	} else {
		value = strings.ReplaceAll(valueStr, "\\,", ",")
	}
	return value
}

// MergeFrom does a key-wise merge between the current map and the passed in map.
// The other map has precedence, and the result will modify the current map.
func (m Map) MergeFrom(other Map) {
	for k, v := range other {
		if vm, ok := v.(Map); ok {
			v = map[string]any(vm)
		}
		if v, ok := v.(map[string]any); ok {
			if bv, ok := m[k]; ok {

				if bv, ok := bv.(map[string]any); ok {
					Map(bv).MergeFrom(v)
					continue
				}
			}
		}
		m[k] = v
	}
}
