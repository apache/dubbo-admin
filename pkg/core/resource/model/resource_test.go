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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseResourceKey(t *testing.T) {
	tests := []struct {
		name        string
		resourceKey string
		wantMesh    string
		wantName    string
	}{
		{name: "mesh and name", resourceKey: "default/demo", wantMesh: "default", wantName: "demo"},
		{name: "name only", resourceKey: "demo", wantMesh: "", wantName: "demo"},
		{name: "empty mesh", resourceKey: "/demo", wantMesh: "", wantName: "demo"},
		{name: "empty name", resourceKey: "default/", wantMesh: "default", wantName: ""},
		{name: "empty key", resourceKey: "", wantMesh: "", wantName: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMesh, gotName := ParseResourceKey(tt.resourceKey)
			assert.Equal(t, tt.wantMesh, gotMesh)
			assert.Equal(t, tt.wantName, gotName)
		})
	}
}
