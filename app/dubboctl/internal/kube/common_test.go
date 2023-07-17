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
	"bytes"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestOverlayObject(t *testing.T) {
	// port and nodePort must be int64 because unstructured.DeepCopy could convert int64
	tests := []struct {
		desc    string
		base    *unstructured.Unstructured
		overlay *unstructured.Unstructured
		want    *unstructured.Unstructured
	}{
		{
			base: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Service",
					"spec": map[string]any{
						"ports": []any{
							map[string]any{
								"port":     int64(8088),
								"nodePort": int64(80),
							},
						},
						"clusterIP": "1.1.1.1",
						"type":      string(v1.ServiceTypeNodePort),
					},
				},
			},
			overlay: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Service",
					"spec": map[string]any{
						"ports": []any{
							map[string]any{
								"port":     int64(8088),
								"nodePort": int64(0),
							},
						},
						"selector": map[string]any{
							"key": "value",
						},
						"type": string(v1.ServiceTypeNodePort),
					},
				},
			},
			want: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Service",
					"spec": map[string]any{
						"ports": []any{
							map[string]any{
								"port":     int64(8088),
								"nodePort": int64(80),
							},
						},
						"selector": map[string]any{
							"key": "value",
						},
						"clusterIP": "1.1.1.1",
						"type":      string(v1.ServiceTypeNodePort),
					},
				},
			},
		},
	}

	for _, test := range tests {
		if err := OverlayObject(test.base, test.overlay); err != nil {
			t.Errorf("OverlayObject failed, err: %s", err)
			return
		}
		resJson, err := test.base.MarshalJSON()
		if err != nil {
			t.Fatalf("res MarshalJSON failed, err: %s", err)
		}
		wantJson, err := test.want.MarshalJSON()
		if err != nil {
			t.Fatalf("want MarshalJSON failed, err: %s", err)
		}
		if !bytes.Equal(resJson, wantJson) {
			t.Error("res obj is not equal to want obj")
			return
		}
	}
}
