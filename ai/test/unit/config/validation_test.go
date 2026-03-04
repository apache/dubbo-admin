package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *AIConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:   "valid config",
			config: &AIConfig{},
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
			errMsg:  "configuration cannot be nil",
		},
		{
			name: "invalid agent max iterations",
			config: &AIConfig{
				Agent: &AgentConfig{
					MaxIterations: -1,
				},
			},
			wantErr: true,
			errMsg:  "agent max iterations must be positive",
		},
		{
			name: "invalid agent buffer size",
			config: &AIConfig{
				Agent: &AgentConfig{
					MaxIterations:          10,
					StageChannelBufferSize: -1,
				},
			},
			wantErr: true,
			errMsg:  "agent stage channel buffer size must be positive",
		},
		{
			name: "empty agent default model",
			config: &AIConfig{
				Agent: &AgentConfig{
					MaxIterations:          10,
					StageChannelBufferSize: 5,
					DefaultModel:           "",
				},
			},
			wantErr: true,
			errMsg:  "agent default model cannot be empty",
		},
		{
			name: "invalid server port",
			config: &AIConfig{
				Agent: &AgentConfig{
					MaxIterations:          10,
					StageChannelBufferSize: 5,
					DefaultModel:           "qwen-max",
				},
				Server: &ServerConfig{
					Port: -1,
				},
			},
			wantErr: true,
			errMsg:  "server port must be between 1 and 65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAgentConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *AgentConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid agent config",
			config: &AgentConfig{
				MaxIterations:          10,
				StageChannelBufferSize: 5,
				DefaultModel:           "qwen-max",
			},
		},
		{
			name:    "nil agent config",
			config:  nil,
			wantErr: true,
			errMsg:  "agent config cannot be nil",
		},
		{
			name: "negative max iterations",
			config: &AgentConfig{
				MaxIterations: -1,
			},
			wantErr: true,
			errMsg:  "max iterations must be positive",
		},
		{
			name: "empty default model",
			config: &AgentConfig{
				MaxIterations: 10,
				DefaultModel:  "",
			},
			wantErr: true,
			errMsg:  "default model cannot be empty",
		},
		{
			name: "valid react config",
			config: &AgentConfig{
				MaxIterations:          10,
				StageChannelBufferSize: 5,
				DefaultModel:           "qwen-max",
				ReAct: &ReActConfig{
					ThinkTemperature:     0.5,
					ActTemperature:       0.7,
					ObserveTemperature:   0.3,
					FeedbackTemperature:  0.2,
					MaxTokens:           2048,
					Timeout:             300,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAgentConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateModelsConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *ModelsConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid models config",
			config: &ModelsConfig{
				DefaultProvider:  "dashscope",
				DefaultModelName: "qwen-max",
			},
		},
		{
			name:    "nil models config",
			config:  nil,
			wantErr: true,
			errMsg:  "models config cannot be nil",
		},
		{
			name: "empty default provider",
			config: &ModelsConfig{
				DefaultProvider: "",
			},
			wantErr: true,
			errMsg:  "default provider cannot be empty",
		},
		{
			name: "empty default model name",
			config: &ModelsConfig{
				DefaultProvider:  "dashscope",
				DefaultModelName: "",
			},
			wantErr: true,
			errMsg:  "default model name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModelsConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateServerConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *ServerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid server config",
			config: &ServerConfig{
				Port:        8888,
				Host:        "0.0.0.0",
				ReadTimeout: 30,
				WriteTimeout: 30,
			},
		},
		{
			name:    "nil server config",
			config:  nil,
			wantErr: true,
			errMsg:  "server config cannot be nil",
		},
		{
			name: "invalid port - too low",
			config: &ServerConfig{
				Port: 0,
			},
			wantErr: true,
			errMsg:  "port must be between 1 and 65535",
		},
		{
			name: "invalid port - too high",
			config: &ServerConfig{
				Port: 65536,
			},
			wantErr: true,
			errMsg:  "port must be between 1 and 65535",
		},
		{
			name: "negative read timeout",
			config: &ServerConfig{
				Port:        8888,
				ReadTimeout: -1,
			},
			wantErr: true,
			errMsg:  "read timeout must be positive",
		},
		{
			name: "negative write timeout",
			config: &ServerConfig{
				Port:         8888,
				ReadTimeout:  30,
				WriteTimeout: -1,
			},
			wantErr: true,
			errMsg:  "write timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServerConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateToolsConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *ToolsConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid tools config",
			config: &ToolsConfig{
				Memory: MemoryConfig{
					DefaultIndex: "test-index",
					TopK:         10,
				},
			},
		},
		{
			name:    "nil tools config",
			config:  nil,
			wantErr: true,
			errMsg:  "tools config cannot be nil",
		},
		{
			name: "empty default index",
			config: &ToolsConfig{
				Memory: MemoryConfig{
					DefaultIndex: "",
				},
			},
			wantErr: true,
			errMsg:  "default memory index cannot be empty",
		},
		{
			name: "invalid top k - too low",
			config: &ToolsConfig{
				Memory: MemoryConfig{
					DefaultIndex: "test-index",
					TopK:         0,
				},
			},
			wantErr: true,
			errMsg:  "top k must be positive",
		},
		{
			name: "invalid top k - too high",
			config: &ToolsConfig{
				Memory: MemoryConfig{
					DefaultIndex: "test-index",
					TopK:         101,
				},
			},
			wantErr: true,
			errMsg:  "top k cannot exceed 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToolsConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateLoggerConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *LoggerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid logger config",
			config: &LoggerConfig{
				Level:  "info",
				Format: "json",
			},
		},
		{
			name:    "nil logger config",
			config:  nil,
			wantErr: true,
			errMsg:  "logger config cannot be nil",
		},
		{
			name: "invalid log level",
			config: &LoggerConfig{
				Level: "invalid",
			},
			wantErr: true,
			errMsg:  "log level must be one of",
		},
		{
			name: "invalid format",
			config: &LoggerConfig{
				Level:  "info",
				Format: "invalid",
			},
			wantErr: true,
			errMsg:  "log format must be 'json' or 'text'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLoggerConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}