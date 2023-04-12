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
	"testing"
)

func TestIsTestYAMLEqual(t *testing.T) {
	tests := []struct {
		desc     string
		golden   string
		result   string
		wantFlag bool
		diff     string
	}{
		{
			desc:     "lines only with whitespaces",
			golden:   "  ",
			result:   "    ",
			wantFlag: true,
		},
		{
			desc:     "non-key:val lines are not equal",
			golden:   "lineG",
			result:   "lineR",
			wantFlag: false,
			diff: `line 1 diff:
--golden--
lineG
--result--
lineR
`,
		},
		{
			desc:     "non-key:val lines are equal",
			golden:   "lineG",
			result:   "lineG",
			wantFlag: true,
		},
		{
			desc:     "key:val lines that vals have different whitespaces",
			golden:   "keyG:valG ",
			result:   "keyG:valG   ",
			wantFlag: true,
		},
		{
			desc:     "key:val lines that keys are not equal",
			golden:   "keyG:valG",
			result:   "keyR:valG",
			wantFlag: false,
			diff: `line 1 diff:
--golden--
keyG:valG
--result--
keyR:valG
`,
		},
		{
			desc:     "key:val lines that vals are not equal",
			golden:   "keyG:valG",
			result:   "keyG:valR",
			wantFlag: false,
			diff: `line 1 diff:
--golden--
keyG:valG
--result--
keyG:valR
`,
		},
		{
			desc:     "key:val lines are equal",
			golden:   "keyG:valG",
			result:   "keyG:valG",
			wantFlag: true,
		},
		{
			desc:     "key:val lines that result line val could match golden line val regularly",
			golden:   "keyG:.*",
			result:   "keyG:valG",
			wantFlag: true,
		},
		{
			desc: "key:val lines are effective but with different indentation",
			golden: `keyG:
  - keyGG: valGG`,
			result: `keyG:
- keyGG: valGG`,
			wantFlag: true,
		},
		{
			desc:     "multi : lines are equal",
			golden:   "keyG:valG:valGG",
			result:   "keyG:valG:valGG",
			wantFlag: true,
		},
		{
			desc: "intact yaml",
			golden: `keyG:valG
keyGG:.*
lineG`,
			result: `keyG:valG
keyGG:valGG
lineG`,
			wantFlag: true,
		},
	}

	for _, test := range tests {
		resFlag, diff, err := TestYAMLEqual(test.golden, test.result)
		if err != nil {
			t.Fatalf("%s failed, err: %s", test.desc, err)
		}
		if resFlag != test.wantFlag {
			t.Errorf("%s test failed, golden:\n%s\nresult:\n%s\n", test.desc, test.golden, test.result)
			return
		}
		if diff != test.diff {
			t.Errorf("%s test failed, got diff:\n%s\nwant diff:\n%s\n", test.desc, diff, test.diff)
		}
	}
}
