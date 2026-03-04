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

package tools

// ToolsConfig defines the tools configuration
type ToolsConfig struct {
	Memory   MemoryConfig   `yaml:"memory"`
	Mock     MockConfig     `yaml:"mock"`
	Internal InternalConfig `yaml:"internal"`
	MCP      MCPConfig      `yaml:"mcp"`
}

// ToolConfig defines the tool configuration
type ToolConfig struct {
	EnableMockTools     bool   `yaml:"enable_mock_tools"`
	EnableInternalTools bool   `yaml:"enable_internal_tools"`
	EnableMCPTools      bool   `yaml:"enable_mcp_tools"`
	MCPHostName         string `yaml:"mcp_host_name"`
	MCPTimeout          int    `yaml:"mcp_timeout"`
	MCPMaxRetries       int    `yaml:"mcp_max_retries"`
}

// MemoryConfig defines the memory configuration
type MemoryConfig struct {
	DefaultIndex  string `yaml:"default_index"`
	TopK          int    `yaml:"top_k"`
	RerankEnabled bool   `yaml:"rerank_enabled"`
	RerankModel   string `yaml:"rerank_model"`
	RerankTopN    int    `yaml:"rerank_top_n"`
}

// MockConfig defines the mock configuration
type MockConfig struct {
	Enabled bool `yaml:"enabled"`
}

// InternalConfig defines the internal configuration
type InternalConfig struct {
	Enabled bool `yaml:"enabled"`
}

// MCPConfig defines the MCP configuration
type MCPConfig struct {
	Enabled bool   `yaml:"enabled"`
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
}

// DefaultToolsConfig returns default tools configuration
func DefaultToolsConfig() ToolsConfig {
	return ToolsConfig{
		Memory:   DefaultMemoryConfig(),
		Mock:     MockConfig{Enabled: true},
		Internal: InternalConfig{Enabled: true},
		MCP:      MCPConfig{Enabled: true, Host: "mcp_host", Port: 8080},
	}
}

// DefaultToolConfig returns default tool configuration
func DefaultToolConfig() *ToolConfig {
	return &ToolConfig{
		EnableMockTools:     true,
		EnableInternalTools: true,
		EnableMCPTools:      true,
		MCPHostName:         "mcp_host",
		MCPTimeout:          30,
		MCPMaxRetries:       3,
	}
}

// DefaultMemoryConfig returns default memory configuration
func DefaultMemoryConfig() MemoryConfig {
	return MemoryConfig{
		DefaultIndex:  "kube-docs",
		TopK:          10,
		RerankEnabled: true,
		RerankModel:   "rerank-v3.5",
		RerankTopN:    2,
	}
}
