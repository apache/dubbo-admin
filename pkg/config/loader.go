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
	"fmt"
	"os"

	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
)

func Load(file string, cfg Config) error {
	return LoadWithOption(file, cfg, false, true, true)
}

func LoadWithOption(file string, cfg Config, strict bool, includeEnv bool, validate bool) error {
	if file == "" {
		return bizerror.New(bizerror.ConfigError, "config file is needed")
	}
	if err := loadFromFile(file, cfg, strict); err != nil {
		return fmt.Errorf("configuration loading failed, %w", err)
	}

	if err := cfg.PreProcess(); err != nil {
		return fmt.Errorf("configuration pre processing failed, %w", err)
	}

	if includeEnv {
		if err := envconfig.Process("", cfg); err != nil {
			return fmt.Errorf("configuration environment processing failed, %w", err)
		}
	}

	if err := cfg.PostProcess(); err != nil {
		return fmt.Errorf("configuration post processing failed, %w", err)
	}

	if validate {
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("configuration validation failed, %w", err)
		}
	}
	return nil
}

func loadFromFile(file string, cfg Config, strict bool) error {
	if _, err := os.Stat(file); err != nil {
		return errors.Errorf("Failed to access configuration file %q", file)
	}
	contents, err := os.ReadFile(file)
	if err != nil {
		return errors.Wrapf(err, "Failed to read configuration from file %q", file)
	}
	if strict {
		err = yaml.UnmarshalStrict(contents, cfg)
	} else {
		err = yaml.Unmarshal(contents, cfg)
	}
	return errors.Wrapf(err, "Failed to parse configuration from file %q", file)
}
