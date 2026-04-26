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

package mcp

import (
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
)

// Config MCP 配置
type Config struct {
	// Enabled 是否启用 MCP 服务器
	Enabled bool `json:"enabled" yaml:"enabled"`
	// ServerName MCP 服务器名称
	ServerName string `json:"serverName" yaml:"serverName"`
	// ServerVersion MCP 服务器版本
	ServerVersion string `json:"serverVersion" yaml:"serverVersion"`
}

// DefaultMCPConfig 返回默认 MCP 配置（默认禁用）
func DefaultMCPConfig() *Config {
	return &Config{
		Enabled:       false,
		ServerName:    "dubbo-admin-mcp",
		ServerVersion: "1.0.0",
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.ServerName == "" {
		return bizerror.New(bizerror.ConfigError, "mcp serverName cannot be empty when enabled")
	}
	if c.ServerVersion == "" {
		return bizerror.New(bizerror.ConfigError, "mcp serverVersion cannot be empty when enabled")
	}
	return nil
}

// Sanitize 清理敏感信息
func (c *Config) Sanitize() {
	// 无需清理
}

// PreProcess 前处理
func (c *Config) PreProcess() error {
	return nil
}

// PostProcess 后处理
func (c *Config) PostProcess() error {
	return nil
}
