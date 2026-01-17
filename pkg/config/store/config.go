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

package store

import (
	"fmt"

	set "github.com/duke-git/lancet/v2/datastructure/set"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/config"
)

var _ config.Config = &Config{}

type Type = string

const (
	Memory   Type = "memory"
	MySQL    Type = "mysql"
	Postgres Type = "postgres"
)

var supportTypes = set.New(Memory, MySQL, Postgres)

// Config defines the ResourceStore configuration
type Config struct {
	config.BaseConfig
	// Type of Store used in Admin
	Type    Type   `json:"type" yaml:"type"`
	Address string `json:"address" yaml:"address"`
}

func (c *Config) Validate() error {
	if !supportTypes.Contain(c.Type) {
		return bizerror.New(bizerror.ConfigError, fmt.Sprintf("unsupported store type: %s", c.Type))
	}
	return nil
}

func DefaultStoreConfig() *Config {
	return &Config{
		Type: Memory,
	}
}
