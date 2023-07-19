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
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"os"
	"strconv"

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
	// there is no easy way to convert yaml to json using gopkg.in/yaml.v2
	return yaml.YAMLToJSON(yamlBytes)
}

func GetStringEnv(name string, defvalue string) string {
	val, ex := os.LookupEnv(name)
	if ex {
		return val
	} else {
		return defvalue
	}
}

func GetIntEnv(name string, defvalue int) int {
	val, ex := os.LookupEnv(name)
	if ex {
		num, err := strconv.Atoi(val)
		if err != nil {
			return defvalue
		} else {
			return num
		}
	} else {
		return defvalue
	}
}

func GetBoolEnv(name string, defvalue bool) bool {
	val, ex := os.LookupEnv(name)
	if ex {
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return defvalue
		} else {
			return boolVal
		}
	} else {
		return defvalue
	}
}

func GetDefaultResourcelockIdentity() string {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	randomBytes := make([]byte, 5)
	_, err = rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}
	randomStr := base32.StdEncoding.EncodeToString(randomBytes)
	return fmt.Sprintf("%s-%s", hostname, randomStr)
}
