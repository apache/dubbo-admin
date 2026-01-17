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
	"os"
	"strings"

	"github.com/apache/dubbo-admin/pkg/config"
)

var _ config.Config = &Config{}

type Type = string

const (
	Memory   Type = "memory"
	MySQL    Type = "mysql"
	Postgres Type = "postgres"
)

// DeploymentMode represents the deployment mode of the store
type DeploymentMode string

const (
	// DeploymentModeSingle represents single instance deployment
	DeploymentModeSingle DeploymentMode = "single"
	// DeploymentModeMasterSlave represents master-slave deployment
	DeploymentModeMasterSlave DeploymentMode = "master-slave"
)

// Config defines the ResourceStore configuration
type Config struct {
	config.BaseConfig
	// Type of Store used in Admin
	Type    Type   `json:"type"`
	Address string `json:"address"`
	// DeploymentMode specifies the deployment mode (single or master-slave)
	DeploymentMode DeploymentMode `json:"deploymentMode"`
	// UseDBIndex forces using database-backed index (overrides auto-detection)
	UseDBIndex *bool `json:"useDbIndex,omitempty"`
}

func DefaultStoreConfig() *Config {
	return &Config{
		Type:           Memory,
		DeploymentMode: detectDeploymentMode(),
		UseDBIndex:     nil, // Auto-detect based on deployment mode
	}
}

// detectDeploymentMode detects the deployment mode from environment
func detectDeploymentMode() DeploymentMode {
	mode := os.Getenv("DUBBO_ADMIN_DB_DEPLOYMENT_MODE")
	if mode != "" {
		mode = strings.ToLower(mode)
		if mode == "master-slave" || mode == "cluster" {
			return DeploymentModeMasterSlave
		}
	}
	return DeploymentModeSingle
}

// ShouldUseDBIndex determines whether to use database-backed index
func (c *Config) ShouldUseDBIndex() bool {
	if c.UseDBIndex != nil {
		return *c.UseDBIndex
	}
	// Auto-detect: use DB index for master-slave deployment
	return c.DeploymentMode == DeploymentModeMasterSlave
}
