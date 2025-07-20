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

package config

import (
	"sigs.k8s.io/yaml"
)

func FromYAML(content []byte, cfg Config) error {
	return yaml.Unmarshal(content, cfg)
}

func ToYAML(cfg Config) ([]byte, error) {
	return yaml.Marshal(cfg)
}

// ToJson converts through YAML, because we only have `yaml` tags on Config.
// This JSON cannot be parsed by json.Unmarshal because durations are marshaled by yaml to pretty form like "1s".
// To change it to simple json.Marshal we need to add `json` tag everywhere.
func ToJson(cfg Config) ([]byte, error) {
	yamlBytes, err := ToYAML(cfg)
	if err != nil {
		return nil, err
	}
	// there is no easy way to convert yaml to json using gopkg.in/yaml.v3
	return yaml.YAMLToJSON(yamlBytes)
}
