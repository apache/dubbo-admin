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
	"reflect"
	"testing"
)

func TestOverlaySetFlags(t *testing.T) {
	tests := []struct {
		base     string
		setFlags []string
		expect   string
	}{
		{
			base: `
spec:
  nacos:
    enabled: true
`,
			setFlags: []string{
				"spec.nacos.enabled=false",
				"spec.nacos.default=true",
				"spec.zookeeper.enabled=true",
			},
			expect: `
spec:
  nacos:
    enabled: false
    default: true
  zookeeper:
    enabled: true
`,
		},
	}
	for _, test := range tests {
		res, err := OverlaySetFlags(test.base, test.setFlags)
		if err != nil {
			t.Fatal(err)
		}

		if reflect.DeepEqual(res, test.expect) {
			t.Errorf("expect %s\n but got %s\n", test.expect, res)
		}
	}
}
