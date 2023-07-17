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
	"bufio"
	"bytes"
	"fmt"
	"strings"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/kylelemons/godebug/diff"
	"sigs.k8s.io/yaml"
)

// IsYAMLEmpty reports whether the YAML string y is logically empty.
func IsYAMLEmpty(y string) bool {
	var yc []string
	for _, l := range strings.Split(y, "\n") {
		yt := strings.TrimSpace(l)
		if !strings.HasPrefix(yt, "#") && !strings.HasPrefix(yt, "---") {
			yc = append(yc, l)
		}
	}
	res := strings.TrimSpace(strings.Join(yc, "\n"))
	return res == "{}" || res == ""
}

// OverlayYAML patches the overlay tree over the base tree and returns the result. All trees are expressed as YAML
// strings.
func OverlayYAML(base, overlay string) (string, error) {
	if strings.TrimSpace(base) == "" {
		return overlay, nil
	}
	if strings.TrimSpace(overlay) == "" {
		return base, nil
	}
	bj, err := yaml.YAMLToJSON([]byte(base))
	if err != nil {
		return "", fmt.Errorf("yamlToJSON error in base: %s\n%s", err, bj)
	}
	oj, err := yaml.YAMLToJSON([]byte(overlay))
	if err != nil {
		return "", fmt.Errorf("yamlToJSON error in overlay: %s\n%s", err, oj)
	}
	if base == "" {
		bj = []byte("{}")
	}
	if overlay == "" {
		oj = []byte("{}")
	}

	merged, err := jsonpatch.MergePatch(bj, oj)
	if err != nil {
		return "", fmt.Errorf("json merge error (%s) for base object: \n%s\n override object: \n%s", err, bj, oj)
	}
	my, err := yaml.JSONToYAML(merged)
	if err != nil {
		return "", fmt.Errorf("jsonToYAML error (%s) for merged object: \n%s", err, merged)
	}

	return string(my), nil
}

// DiffYAML accepts two single yaml(not with "---" yaml separator).
// Returns a string containing a line-by-line unified diff of the linewise
// changes required to make A into B.  Each line is prefixed with '+', '-', or
// ' ' to indicate if it should be added, removed, or is correct respectively.
//
// eg:
// a: key: val
//
//	key1: val1
//
// b: key: val
//
//	key1: val2
//
// return: key: val
//
//	-key1: val1
//	+key1: val2
func DiffYAML(a, b string) (string, error) {
	// Unmarshal and Marshal to keep the order of the keys consistent
	aMap, bMap := make(map[string]any), make(map[string]any)
	if err := yaml.Unmarshal([]byte(a), &aMap); err != nil {
		return "", err
	}
	if err := yaml.Unmarshal([]byte(b), &bMap); err != nil {
		return "", err
	}
	var aYaml, bYaml string
	if len(aMap) != 0 {
		aBytes, err := yaml.Marshal(aMap)
		if err != nil {
			return "", err
		}
		aYaml = string(aBytes)
	}
	if len(bMap) != 0 {
		bBytes, err := yaml.Marshal(bMap)
		if err != nil {
			return "", err
		}
		bYaml = string(bBytes)
	}

	return diff.Diff(aYaml, bYaml), nil
}

// SplitYAML uses yaml separator that begins with "---" to split yaml file.
func SplitYAML(y string) ([]string, error) {
	var buf bytes.Buffer
	var segments []string
	scanner := bufio.NewScanner(strings.NewReader(y))
	for scanner.Scan() {
		line := scanner.Text()
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
		segments = append(segments, buf.String()+"\n")
	}
	return segments, nil
}

// JoinYAML use "---" to join yaml segments.
func JoinYAML(segs []string) string {
	var resBuilder strings.Builder

	for _, seg := range segs {
		seg = strings.TrimSpace(seg)
		// todo: support yaml separator begin with "---"
		if !strings.HasSuffix(seg, "---") {
			resBuilder.WriteString(seg + "\n---\n")
		} else {
			resBuilder.WriteString(seg + "\n")
		}
	}
	return resBuilder.String()
}
