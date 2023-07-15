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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestCompareObjects(t *testing.T) {
	testNamespace := "test"
	tests := []struct {
		desc  string
		objsA []*Object
		objsB []*Object
		diff  string
		add   string
		err   string
	}{
		{
			desc: "objects could be parsed correctly",
			objsA: []*Object{
				{
					Namespace: testNamespace,
					Kind:      "kind1",
					Name:      "name1",
					yamlStr: `key1: val1
key2: val2`,
				},
				{
					Namespace: testNamespace,
					Kind:      "kind2",
					Name:      "name2",
					yamlStr:   `key1: val1`,
				},
				{
					Namespace: testNamespace,
					Kind:      "kind3",
					Name:      "name3",
					yamlStr:   `key1: val1`,
				},
			},
			objsB: []*Object{
				{
					Namespace: testNamespace,
					Kind:      "kind1",
					Name:      "name1",
					yamlStr:   `key1: val`,
				},
				{
					Namespace: testNamespace,
					Kind:      "kind2",
					Name:      "name2",
					yamlStr: `key1: val1
key2: val2`,
				},
				{
					Namespace: testNamespace,
					Kind:      "kind4",
					Name:      "name4",
					yamlStr:   `key1: val`,
				},
			},
			diff: `Namespace:Kind:Name=>test:kind1:name1 diff:
-key1: val1
-key2: val2
+key1: val
 
------
Namespace:Kind:Name=>test:kind2:name2 diff:
 key1: val1
+key2: val2
 
------
`,
			add: `Namespace:Kind:Name=>test:kind3:name3 in previous addition:
-key1: val1
 
------
Namespace:Kind:Name=>test:kind4:name4 in next addition:
-key1: val
 
------
`,
		},
		{
			desc: "objects with wrong format",
			objsA: []*Object{
				{
					Namespace: testNamespace,
					Kind:      "kind1",
					Name:      "name1",
					yamlStr: `key1: val1
  key2: val2`,
				},
				{
					Namespace: testNamespace,
					Kind:      "kind2",
					Name:      "name2",
					yamlStr: `key1: val1
  key2: val2`,
				},
			},
			objsB: []*Object{
				{
					Namespace: testNamespace,
					Kind:      "kind1",
					Name:      "name1",
					yamlStr: `key1: val1
key2: val2`,
				},
				{
					Namespace: testNamespace,
					Kind:      "kind3",
					Name:      "name3",
					yamlStr: `key1: val1
 key2: val2`,
				},
			},
			err: `Namespace:Kind:Name=>test:kind1:name1 parse failed, err:
error converting YAML to JSON: yaml: line 2: mapping values are not allowed in this context
------
Namespace:Kind:Name=>test:kind2:name2 in previous section parse failed, err:
error converting YAML to JSON: yaml: line 2: mapping values are not allowed in this context
------
Namespace:Kind:Name=>test:kind3:name3 in next section parse failed, err:
error converting YAML to JSON: yaml: line 2: mapping values are not allowed in this context
------
`,
		},
	}
	for _, test := range tests {
		diffRes, addRes, errRes := CompareObjects(test.objsA, test.objsB)
		if diffRes != test.diff {
			t.Errorf("want diff:\n%s\nbut got:\n%s\n", test.diff, diffRes)
			return
		}
		if addRes != test.add {
			t.Errorf("want add:\n%s\nbut got:\n%s\n", test.add, addRes)
			return
		}
		if errRes != test.err {
			t.Errorf("want err:\n%s\nbut got:\n%s\n", test.err, errRes)
			return
		}
	}
}

func TestCompareObject(t *testing.T) {
	tests := []struct {
		desc    string
		objA    *Object
		objB    *Object
		diff    string
		wantErr bool
	}{
		{
			desc: "objects could be parsed correctly",
			objA: &Object{
				yamlStr: `key1: val1
key2: val2`,
			},
			objB: &Object{
				yamlStr: `key1: val1
key2: val3`,
			},
			diff: `key1: val1
-key2: val2
+key2: val3`,
		},
		{
			desc: "objects with wrong format",
			objA: &Object{
				yamlStr: `key1: val1
  key2: val2`,
			},
			objB: &Object{
				yamlStr: `key1: val1
key2: val3`,
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			diff, err := CompareObject(test.objA, test.objB)
			if err != nil {
				if !test.wantErr {
					t.Errorf("execute failed, err: %s", err)
				}
			} else {
				if test.wantErr {
					t.Errorf("execution expected to fail, but succeed")
				} else {
					if strings.TrimSpace(diff) != test.diff {
						t.Errorf("want:\n%s\nbut got:\n%s", test.diff, diff)
					}
				}
			}
		})
	}
}
