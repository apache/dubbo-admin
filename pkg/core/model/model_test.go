/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package model

import (
	"fmt"
	"testing"
	"time"

	network "github.com/apache/dubbo-admin/api/resource/v1alpha1"
	"github.com/google/go-cmp/cmp"
)

func TestDeepCopy(t *testing.T) {
	cfg := Config{
		Meta: Meta{
			Name:              "name1",
			Namespace:         "zzz",
			CreationTimestamp: time.Now(),
			Labels:            map[string]string{"app": "test-app"},
			Annotations:       map[string]string{"test-annotations": "3"},
		},
		Spec: &network.TagRoute{},
	}

	copied := cfg.DeepCopy()

	if diff := cmp.Diff(copied, cfg); diff != "" {
		t.Fatalf("cloned config is not identical: %v", diff)
	}

	copied.Labels["app"] = "cloned-app"
	copied.Annotations["test-annotations"] = "0"
	if cfg.Labels["app"] == copied.Labels["app"] ||
		cfg.Annotations["test-annotations"] == copied.Annotations["test-annotations"] {
		t.Fatalf("Did not deep copy labels and annotations")
	}

	// change the copied tagroute to see if the original config is not effected
	copiedTagRoute := copied.Spec.(*network.TagRoute)
	copiedTagRoute.Tags = []*network.Tag{
		{
			Name: "test",
		},
	}

	tagRoute := cfg.Spec.(*network.TagRoute)
	if tagRoute.Tags != nil {
		t.Errorf("Original gateway is mutated")
	}
}

func TestDeepCopyTypes(t *testing.T) {
	cases := []struct {
		input  Spec
		modify func(c Spec) Spec
		option cmp.Options
	}{
		{
			input: &network.TagRoute{
				Tags: []*network.Tag{
					{
						Addresses: []string{"lxy"},
					},
				},
			},
			modify: func(c Spec) Spec {
				route := c.(*network.TagRoute)
				route.Tags[0].Addresses = []string{"zyq"}
				return c
			},
			option: nil,
		},
	}
	for _, tt := range cases {
		t.Run(fmt.Sprintf("%T", tt.input), func(t *testing.T) {
			cpy := DeepCopy(tt.input)
			if diff := cmp.Diff(tt.input, cpy, tt.option); diff != "" {
				t.Fatalf("Type was %T now is %T. Diff: %v", tt.input, cpy, diff)
			}
			changed := tt.modify(tt.input)
			if cmp.Equal(cpy, changed, tt.option) {
				t.Fatalf("deep copy allowed modification")
			}
		})
	}
}
