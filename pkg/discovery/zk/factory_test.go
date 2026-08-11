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

package zk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZKConfigNameSupportsLegacyAndGroupedConfigPaths(t *testing.T) {
	tests := []struct {
		name     string
		nodePath string
		wantName string
		wantOK   bool
	}{
		{
			name:     "legacy direct config key",
			nodePath: "/dubbo/config/demo-provider.condition-router",
			wantName: "demo-provider.condition-router",
			wantOK:   true,
		},
		{
			name:     "dubbo-go grouped config key",
			nodePath: "/dubbo/config/dubbo/org.apache.demo.DemoService:1.0.0:demo.condition-router",
			wantName: "org.apache.demo.DemoService:1.0.0:demo.condition-router",
			wantOK:   true,
		},
		{
			name:     "config root",
			nodePath: "/dubbo/config",
			wantOK:   false,
		},
		{
			name:     "group root",
			nodePath: "/dubbo/config/dubbo",
			wantOK:   true,
			wantName: "dubbo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotOK := zkConfigName(tt.nodePath)
			assert.Equal(t, tt.wantOK, gotOK)
			assert.Equal(t, tt.wantName, gotName)
		})
	}
}
