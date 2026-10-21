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

package v1alpha1

import (
	"sigs.k8s.io/yaml"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

// MarshalRuleSpec encodes a rule for storage in a configuration center.
// Router booleans must remain explicit: their protobuf JSON tags omit false,
// while a runtime may treat an omitted flag differently from explicit false.
func MarshalRuleSpec(spec coremodel.ResourceSpec) ([]byte, error) {
	switch rule := spec.(type) {
	case *meshproto.ConditionRoute:
		if rule != nil {
			return yaml.Marshal(struct {
				*meshproto.ConditionRoute
				Enabled bool `json:"enabled"`
				Force   bool `json:"force"`
				Runtime bool `json:"runtime"`
			}{rule, rule.Enabled, rule.Force, rule.Runtime})
		}
	case *meshproto.TagRoute:
		if rule != nil {
			return yaml.Marshal(struct {
				*meshproto.TagRoute
				Enabled bool `json:"enabled"`
				Force   bool `json:"force"`
				Runtime bool `json:"runtime"`
			}{rule, rule.Enabled, rule.Force, rule.Runtime})
		}
	}
	return yaml.Marshal(spec)
}
