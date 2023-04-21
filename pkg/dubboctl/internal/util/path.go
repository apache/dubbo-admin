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

package util

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	PathSeparator        = "."
	pathSeparatorRune    = '.'
	EscapedPathSeparator = "\\" + PathSeparator
	kvSeparatorRune      = ':'
	InsertIndex          = -1
)

// ValidKeyRegex is a regex for a valid path key element.
var ValidKeyRegex = regexp.MustCompile("^[a-zA-Z0-9_-]*$")

// Path is a path in slice form.
type Path []string

// PathFromString converts a string path of form a.b.c to a string slice representation.
func PathFromString(path string) Path {
	path = filepath.Clean(path)
	path = strings.TrimPrefix(path, PathSeparator)
	path = strings.TrimSuffix(path, PathSeparator)
	pv := splitEscaped(path, pathSeparatorRune)
	var r []string
	for _, str := range pv {
		if str != "" {
			str = strings.ReplaceAll(str, EscapedPathSeparator, PathSeparator)
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

// String converts a string slice path representation of form ["a", "b", "c"] to a string representation like "a.b.c".
func (p Path) String() string {
	return strings.Join(p, PathSeparator)
}

func (p Path) Equals(p2 Path) bool {
	if len(p) != len(p2) {
		return false
	}
	for i, pp := range p {
		if pp != p2[i] {
			return false
		}
	}
	return true
}

// ToYAMLPath converts a path string to path such that the first letter of each path element is lower case.
func ToYAMLPath(path string) Path {
	p := PathFromString(path)
	for i := range p {
		p[i] = firstCharToLowerCase(p[i])
	}
	return p
}

func firstCharToLowerCase(s string) string {
	return strings.ToLower(s[0:1]) + s[1:]
}

// splitEscaped splits a string using the rune r as a separator. It does not split on r if it's prefixed by \.
func splitEscaped(s string, r rune) []string {
	var prev rune
	if len(s) == 0 {
		return []string{}
	}
	prevIdx := 0
	var out []string
	for i, c := range s {
		if c == r && (i == 0 || (i > 0 && prev != '\\')) {
			out = append(out, s[prevIdx:i])
			prevIdx = i + 1
		}
		prev = c
	}
	out = append(out, s[prevIdx:])
	return out
}

// RemoveBrackets removes the [] around pe and returns the resulting string. It returns false if pe is not surrounded
// by [].
func RemoveBrackets(pe string) (string, bool) {
	if !strings.HasPrefix(pe, "[") || !strings.HasSuffix(pe, "]") {
		return "", false
	}
	return pe[1 : len(pe)-1], true
}

// PathKV returns the key and value string parts of the entire key/value path element.
// It returns an error if pe is not a key/value path element.
func PathKV(pe string) (k, v string, err error) {
	if !IsKVPathElement(pe) {
		return "", "", fmt.Errorf("%s is not a valid key:value path element", pe)
	}
	pe, _ = RemoveBrackets(pe)
	kv := splitEscaped(pe, kvSeparatorRune)
	return kv[0], kv[1], nil
}

// PathV returns the value string part of the entire value path element.
// It returns an error if pe is not a value path element.
func PathV(pe string) (string, error) {
	// For :val, return the value only
	if IsVPathElement(pe) {
		v, _ := RemoveBrackets(pe)
		return v[1:], nil
	}

	// For key:val, return the whole thing
	v, _ := RemoveBrackets(pe)
	if len(v) > 0 {
		return v, nil
	}
	return "", fmt.Errorf("%s is not a valid value path element", pe)
}

// PathN returns the index part of the entire value path element.
// It returns an error if pe is not an index path element.
func PathN(pe string) (int, error) {
	if !IsNPathElement(pe) {
		return -1, fmt.Errorf("%s is not a valid index path element", pe)
	}
	v, _ := RemoveBrackets(pe)
	return strconv.Atoi(v)
}

// IsVPathElement report whether pe is a value path element.
func IsVPathElement(pe string) bool {
	pe, ok := RemoveBrackets(pe)
	if !ok {
		return false
	}

	return len(pe) > 1 && pe[0] == ':'
}

// IsNPathElement report whether pe is an index path element.
func IsNPathElement(pe string) bool {
	pe, ok := RemoveBrackets(pe)
	if !ok {
		return false
	}

	n, err := strconv.Atoi(pe)
	return err == nil && n >= InsertIndex
}

// IsValidPathElement reports whether pe is a valid path element.
func IsValidPathElement(pe string) bool {
	return ValidKeyRegex.MatchString(pe)
}

// IsKVPathElement report whether pe is a key/value path element.
func IsKVPathElement(pe string) bool {
	pe, ok := RemoveBrackets(pe)
	if !ok {
		return false
	}

	kv := splitEscaped(pe, kvSeparatorRune)
	if len(kv) != 2 || len(kv[0]) == 0 || len(kv[1]) == 0 {
		return false
	}
	return IsValidPathElement(kv[0])
}
