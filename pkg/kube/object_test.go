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
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseObjectFromManifest(t *testing.T) {
	tests := []struct {
		desc       string
		manifest   string
		wantErr    bool
		assertFunc func(obj *Object)
	}{
		{
			desc: "parse object from correct format manifest",
			manifest: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config
  namespace: dubbo-system
data:
  key1: data1`,
			assertFunc: func(obj *Object) {
				assert.Equal(t, "ConfigMap", obj.Kind)
				assert.Equal(t, "config", obj.Name)
				assert.Equal(t, "dubbo-system", obj.Namespace)
				assert.Equal(t, map[string]interface{}{"key1": "data1"}, obj.Unstructured().Object["data"])
			},
		},
		{
			desc:     "parse object from incorrect format manifest",
			manifest: "wrong format###",
			wantErr:  true,
		},
	}

	for _, test := range tests {
		obj, err := ParseObjectFromManifest(test.manifest)
		if err != nil {
			if !test.wantErr {
				t.Errorf("%s err: %s", test.desc, err)
			}
		} else {
			test.assertFunc(obj)
		}
	}
}

func TestParseObjectsFromManifest(t *testing.T) {
	tests := []struct {
		desc          string
		manifest      string
		wantErr       bool
		continueOnErr bool
		expectedObjs  int
	}{
		{
			desc:         "parse objects from manifest separated by yaml separator",
			expectedObjs: 3,
			manifest: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
  namespace: dubbo-system
data:
  key1: data1
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2
  namespace: dubbo-system
data:
  key2: data2
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config3
  namespace: dubbo-system
data:
  key3: data3`,
		},
		{
			desc:          "parse partial objects from manifest containing wrong format yaml when setting continueOnErr",
			continueOnErr: true,
			expectedObjs:  2,
			manifest: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
  namespace: dubbo-system
data:
  key1: data1
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2
  namespace: dubbo-system
data:
  key2: data2
---
wrong format###`,
		},
		{
			desc:    "parse objects failed from manifest containing wrong format yaml",
			wantErr: true,
			manifest: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
  namespace: dubbo-system
data:
  key1: data1
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: config2
  namespace: dubbo-system
data:
  key2: data2
---
wrong format###`,
		},
	}

	for _, test := range tests {
		objs, err := ParseObjectsFromManifest(test.manifest, test.continueOnErr)
		if err != nil {
			if !test.wantErr {
				t.Errorf("%s err: %s", test.desc, err)
			}
		}
		if len(objs) != test.expectedObjs {
			t.Errorf("expected %d obj, but got %d obj", test.expectedObjs, len(objs))
		}
	}
}
