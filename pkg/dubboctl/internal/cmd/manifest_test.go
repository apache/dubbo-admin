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

package cmd

import (
	"reflect"
	"testing"
)

func TestGenerateValues(t *testing.T) {
	tests := []struct {
		args       *ManifestGenerateArgs
		expectVals string
	}{
		{
			args: &ManifestGenerateArgs{
				FileNames:    nil,
				ChartsPath:   "../../../deploy/charts/admin-stack/charts",
				ProfilesPath: "../../../deploy/profiles",
				OutputPath:   "",
				SetFlags:     nil,
			},
			expectVals: `
apiVersion: dubbo.apache.org/v1alpha1
kind: DubboOperator
metadata:
  namespace: dubbo-system
spec:
  componentsMeta:
    zookeeper:
      enabled: true
      namespace: dubbo-system
  profile: default
`,
		},
	}

	for _, test := range tests {
		op, vals, err := generateValues(test.args)
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(vals, test.expectVals) {
			t.Errorf("expect: %s\n res: %s", test.expectVals, vals)
		}
		if err := generateManifests(test.args, op); err != nil {
			t.Error(err)
		}
	}
}
