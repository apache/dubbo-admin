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

package tools

import (
	"dubbo-admin-ai/runtime"
	"fmt"

	"gopkg.in/yaml.v3"
)

// ToolsFactory creates a tools component (explicit registration, no init)
func ToolsFactory(spec *yaml.Node) (runtime.Component, error) {
	var cfg ToolConfig
	if err := spec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode tools spec: %w", err)
	}

	return NewToolsComponent(cfg)
}
