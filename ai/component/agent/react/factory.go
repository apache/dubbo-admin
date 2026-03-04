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

package react

import (
	"dubbo-admin-ai/runtime"
	"fmt"

	"gopkg.in/yaml.v3"
)

// AgentFactory creates an agent component (explicit registration, no init)
func AgentFactory(spec *yaml.Node) (runtime.Component, error) {
	var cfg AgentSpec
	if err := spec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode agent spec: %w", err)
	}

	return NewAgentComponent(
		cfg.AgentType,
		cfg.Model,
		cfg.PromptBasePath,
		cfg.MaxIterations,
		cfg.StageChannelBufferSize,
		cfg.MCPHostName,
		cfg.Stages,
	)
}
