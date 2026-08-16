/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License); you may not use this file except in compliance with
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

package testutils

import (
	"gopkg.in/yaml.v3"
)

// ConfigFixture 提供各组件的测试配置
type ConfigFixture struct{}

// ValidLoggerConfig 返回有效的Logger配置
func (f *ConfigFixture) ValidLoggerConfig(level string) *yaml.Node {
	var node yaml.Node
	node.Encode(map[string]any{
		"level": level,
	})
	return &node
}

// InvalidLoggerConfig 返回无效的Logger配置（空配置）
func (f *ConfigFixture) InvalidLoggerConfig() *yaml.Node {
	var node yaml.Node
	node.Encode(map[string]any{})
	return &node
}

// ValidMemoryConfig 返回有效的Memory配置
func (f *ConfigFixture) ValidMemoryConfig() *yaml.Node {
	var node yaml.Node
	node.Encode(map[string]any{
		"history_key": "chat_history",
		"max_turns":   100,
	})
	return &node
}

// InvalidMemoryConfig 返回无效的Memory配置
func (f *ConfigFixture) InvalidMemoryConfig() *yaml.Node {
	var node yaml.Node
	node.Encode(map[string]any{
		"max_turns": "invalid", // 错误的类型
	})
	return &node
}

// ValidModelsConfig 返回有效的Models配置
func (f *ConfigFixture) ValidModelsConfig() *yaml.Node {
	var node yaml.Node
	node.Encode(map[string]any{
		"default_provider":  "dashscope",
		"default_model":     "dashscope/qwen-max",
		"default_embedding": "dashscope/qwen3-embedding",
		"providers": map[string]any{
			"dashscope": map[string]any{
				"api_key":  "test-key-${DASHSCOPE_API_KEY}",
				"base_url": "https://dashscope.aliyuncs.com/compatible-mode/v1",
				"models": []map[string]any{
					{
						"name": "qwen-max",
						"key":  "qwen-max",
						"type": "chat",
					},
				},
				"embedders": []map[string]any{
					{
						"name":       "qwen3-embedding",
						"key":        "qwen3-embedding",
						"type":       "text",
						"dimensions": 1024,
					},
				},
			},
		},
	})
	return &node
}

// ModelsConfigWithEmptyAPIKey 返回包含空API Key的Models配置
func (f *ConfigFixture) ModelsConfigWithEmptyAPIKey() *yaml.Node {
	var node yaml.Node
	node.Encode(map[string]any{
		"default_model":     "dashscope/qwen-max",
		"default_embedding": "dashscope/qwen3-embedding",
		"providers": map[string]any{
			"dashscope": map[string]any{
				"api_key":  "", // 空API Key
				"base_url": "https://dashscope.aliyuncs.com/compatible-mode/v1",
				"models": []map[string]any{
					{
						"name": "qwen-max",
						"key":  "qwen-max",
						"type": "chat",
					},
				},
			},
		},
	})
	return &node
}

// ModelsConfigWithMultipleProviders 返回包含多个Provider的Models配置
func (f *ConfigFixture) ModelsConfigWithMultipleProviders() *yaml.Node {
	var node yaml.Node
	node.Encode(map[string]any{
		"default_model":     "dashscope/qwen-max",
		"default_embedding": "dashscope/qwen3-embedding",
		"providers": map[string]any{
			"dashscope": map[string]any{
				"api_key":  "test-dashscope-key",
				"base_url": "https://dashscope.aliyuncs.com/compatible-mode/v1",
				"models": []map[string]any{
					{"name": "qwen-max", "key": "qwen-max", "type": "chat"},
				},
				"embedders": []map[string]any{
					{"name": "qwen3-embedding", "key": "qwen3-embedding", "dimensions": 1024},
				},
			},
			"gemini": map[string]any{
				"api_key":  "", // Gemini需要特殊处理
				"base_url": "https://generativelanguage.googleapis.com",
				"models": []map[string]any{
					{"name": "gemini-pro", "key": "gemini-pro", "type": "chat"},
				},
			},
		},
	})
	return &node
}

// ValidToolsConfig 返回有效的Tools配置
func (f *ConfigFixture) ValidToolsConfig() *yaml.Node {
	var node yaml.Node
	node.Encode(map[string]any{
		"enable_mock_tools":     true,
		"enable_internal_tools": false,
		"enable_mcp_tools":      false,
	})
	return &node
}

// ToolsConfigWithAllEnabled 返回所有工具都启用的配置
func (f *ConfigFixture) ToolsConfigWithAllEnabled() *yaml.Node {
	var node yaml.Node
	node.Encode(map[string]any{
		"enable_mock_tools":     true,
		"enable_internal_tools": true,
		"enable_mcp_tools":      true,
	})
	return &node
}

// ValidRAGConfig 返回有效的RAG配置
func (f *ConfigFixture) ValidRAGConfig() *yaml.Node {
	var node yaml.Node
	node.Encode(map[string]any{
		"loader": map[string]any{
			"type": "pdf",
		},
		"splitter": map[string]any{
			"type":          "recursive",
			"chunk_size":    1000,
			"chunk_overlap": 200,
		},
		"indexer": map[string]any{
			"type":       "pinecone",
			"index_name": "test-index",
			"dimension":  1024,
		},
		"retriever": map[string]any{
			"type":  "vector",
			"top_k": 10,
		},
	})
	return &node
}

// ValidAgentConfig 返回有效的Agent配置
func (f *ConfigFixture) ValidAgentConfig() *yaml.Node {
	var node yaml.Node
	node.Encode(map[string]any{
		"agent_type":          "react",
		"model":               "qwen-max",
		"prompt_base_path":    "./prompts",
		"prompt_file":         "agentReasonAct.txt",
		"max_iterations":      3,
		"channel_buffer_size": 5,
		"temperature":         0.7,
		"top_p":               0.9,
		"max_tokens":          3000,
		"timeout":             90,
	})
	return &node
}

// ValidServerConfig 返回有效的Server配置
func (f *ConfigFixture) ValidServerConfig() *yaml.Node {
	var node yaml.Node
	node.Encode(map[string]any{
		"host": "localhost",
		"port": 8080,
	})
	return &node
}

// ServerConfigWithCustomPort 返回自定义端口的Server配置
func (f *ConfigFixture) ServerConfigWithCustomPort(port int) *yaml.Node {
	var node yaml.Node
	node.Encode(map[string]any{
		"host": "0.0.0.0",
		"port": port,
	})
	return &node
}
