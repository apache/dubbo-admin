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

package manifest

import (
	"testing"
)

func TestOverlaySetFlags(t *testing.T) {
	testBase := `spec:
  components:
    nacos:
      enabled: true
`
	tests := []struct {
		desc     string
		base     string
		setFlags []string
		expect   string
	}{
		{
			desc: "create missing leaf",
			base: testBase,
			setFlags: []string{
				"spec.components.nacos.replicas=2",
			},
			expect: `spec:
  components:
    nacos:
      enabled: true
      replicas: 2
`,
		},
		{
			desc: "create missing object",
			base: testBase,
			setFlags: []string{
				"spec.components.zookeeper.enabled=true",
				"spec.components.zookeeper.replicas=2",
			},
			expect: `spec:
  components:
    nacos:
      enabled: true
    zookeeper:
      enabled: true
      replicas: 2
`,
		},
		{
			desc: "replace leaf",
			base: testBase,
			setFlags: []string{
				"spec.components.nacos.enabled=false",
			},
			expect: `spec:
  components:
    nacos:
      enabled: false
`,
		},
	}
	for _, test := range tests {
		res, err := OverlaySetFlags(test.base, test.setFlags)
		if err != nil {
			t.Fatal(err)
		}
		if res != test.expect {
			t.Errorf("expect:\n%s\nbut got:\n%s\n", test.expect, res)
		}
	}
}
