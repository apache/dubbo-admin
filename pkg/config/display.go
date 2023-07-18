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
	"encoding/json"
	"os"
	"reflect"

	"gopkg.in/yaml.v2"
)

func ConfigForDisplay(cfg Config) (Config, error) {
	// copy config so we don't override values, because nested structs in config are pointers
	newCfg, err := copyConfig(cfg)
	if err != nil {
		return nil, err
	}
	newCfg.Sanitize()
	return newCfg, nil
}

func copyConfig(cfg Config) (Config, error) {
	cfgBytes, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	newCfg := reflect.New(reflect.TypeOf(cfg).Elem()).Interface().(Config)
	if err := json.Unmarshal(cfgBytes, newCfg); err != nil {
		return nil, err
	}
	return newCfg, nil
}

func DumpToFile(filename string, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, b, 0o666)
}
