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
	"os"
	"path/filepath"

	"github.com/apache/dubbo-admin/pkg/core/logger"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func Load(file string, cfg Config) error {
	return LoadWithOption(file, cfg, true)
}

func LoadWithOption(file string, cfg Config, validate bool) error {
	if file == "" {
		file = Conf
		if envPath := os.Getenv(confPathKey); envPath != "" {
			file = envPath
		}
	}
	path, err := filepath.Abs(file)
	logger.Info("config path: ", path)
	if err != nil {
		path = filepath.Clean(file)
	}
	if _, err := os.Stat(file); err != nil {
		return errors.Errorf("Failed to access configuration file %q", file)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(content, cfg)
	if err != nil {
		logger.Errorf("Invalid configuration: \n %s", content)
		panic(err)
	}
	if validate {
		if err := cfg.Validate(); err != nil {
			return errors.Wrapf(err, "Invalid configuration")
		}
	}
	return nil
}
