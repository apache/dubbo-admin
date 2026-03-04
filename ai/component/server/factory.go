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

package server

import (
	"dubbo-admin-ai/runtime"
	"fmt"

	"gopkg.in/yaml.v3"
)

// ServerFactory component factory function (explicit registration, does not use init)
func ServerFactory(spec *yaml.Node) (runtime.Component, error) {
	if spec == nil {
		return nil, fmt.Errorf("spec is nil")
	}

	var cfg ServerSpec
	if err := spec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode server spec: %w", err)
	}

	return NewServerComponent(
		cfg.Port,
		cfg.Host,
		cfg.Debug,
		cfg.CORSOrigins,
		cfg.ReadTimeout,
		cfg.WriteTimeout,
	)
}
