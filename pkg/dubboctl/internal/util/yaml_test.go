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

package util

import (
	"strings"
	"testing"
)

func TestDiffYAML(t *testing.T) {
	tests := []struct {
		desc    string
		a       string
		b       string
		want    string
		wantErr bool
	}{
		{
			desc: "same yaml",
			a:    `key: val`,
			b:    `key: val`,
			want: "",
		},
		{
			desc: "same yaml with different order",
			a: `key: val
key1: val1`,
			b: `key1: val1
key: val`,
			want: "",
		},
		{
			desc: "different yaml with same lengths",
			a: `key: val
key1: val1`,
			b: `key: val
key1: val2`,
			want: `key: val
-key1: val1
+key1: val2`,
		},
		{
			desc: "different yaml and a is longer than b",
			a: `key: val
key1: val1`,
			b: `key: val1`,
			want: `-key: val
-key1: val1
+key: val1`,
		},
		{
			desc: "different yaml and b is longer than a",
			a:    `key: val1`,
			b: `key: val
key1: val1`,
			want: `-key: val1
+key: val
+key1: val1`,
		},
		{
			desc: "a is malformed",
			a: `key: val
  key1: val1`,
			b:       `key: val`,
			wantErr: true,
		},
		{
			desc: "b is malformed",
			a:    `key: val`,
			b: `key: val
  key1: val1`,
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			res, err := DiffYAML(test.a, test.b)
			if err != nil {
				if !test.wantErr {
					t.Errorf("YAMLDiff failed, err: %s", err)
					return
				}
			}
			if strings.TrimSpace(res) != strings.TrimSpace(test.want) {
				t.Errorf("got res:\n%s\nbut want:\n%s\n", res, test.want)
				return
			}
		})
	}
}

func TestSplitYAML(t *testing.T) {
	tests := []struct {
		desc  string
		input string
		want  []string
	}{
		{
			desc:  "yaml without separator",
			input: `key: val`,
			want: []string{
				`key: val`,
			},
		},
		{
			desc: "yaml ends with separator",
			input: `key: val
---`,
			want: []string{
				`key: val`,
			},
		},
		{
			desc: "yaml could be separated",
			input: `key: val
---
key1: val1`,
			want: []string{
				`key: val`,
				`key1: val1`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			res, err := SplitYAML(test.input)
			if err != nil {
				t.Errorf("SplitYAML failed, err: %s", err)
				return
			}

			for i, seg := range res {
				if i >= len(test.want) || strings.TrimSpace(seg) != strings.TrimSpace(test.want[i]) {
					t.Errorf("want:\n%v\nbut got:\n%v\n", test.want, res)
					return
				}
			}
		})
	}
}

func TestJoinYAML(t *testing.T) {
	tests := []struct {
		desc  string
		input []string
		want  string
	}{
		{
			desc: "yaml ends with separator",
			input: []string{
				`key: val
---`,
				`key1: val1
---`,
			},
			want: `key: val
---
key1: val1
---`,
		},
		{
			desc: "yaml not end with separator",
			input: []string{
				`key: val`,
				`key1: val1`,
			},
			want: `key: val
---
key1: val1
---`,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			res := JoinYAML(test.input)
			if strings.TrimSpace(res) != strings.TrimSpace(test.want) {
				t.Errorf("want:\n%s\nbut got:\n%s\n", test.want, res)
			}
		})
	}
}
