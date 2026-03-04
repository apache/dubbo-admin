package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromFile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	// Create test config content
	configContent := `
agent:
  max_iterations: 15
  stage_channel_buffer_size: 10
  default_model: "gpt-4"
  mcp_host_name: "test-host"
  prompt_base_path: "./test-prompts"
  react:
    think_temperature: 0.3
    act_temperature: 0.4
    observe_temperature: 0.5
    feedback_temperature: 0.6
    max_tokens: 1024
    timeout: 60

models:
  default_provider: "openai"
  default_model_name: "gpt-4"
  default_embedding: "text-embedding-ada-002"

server:
  port: 9999
  host: "127.0.0.1"
  read_timeout: 60
  write_timeout: 60
  cors_origins: ["http://localhost:3000"]

tools:
  memory:
    default_index: "test-index"
    top_k: 20
    rerank_enabled: true
    rerank_model: "test-rerank"
    rerank_top_n: 5

logger:
  level: "debug"
  format: "text"
`

	// Write config to file
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load the configuration
	cfg, err := LoadFromFile(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify loaded values
	assert.Equal(t, 15, cfg.Agent.MaxIterations)
	assert.Equal(t, 10, cfg.Agent.StageChannelBufferSize)
	assert.Equal(t, "gpt-4", cfg.Agent.DefaultModel)
	assert.Equal(t, "test-host", cfg.Agent.MCPHostName)
	assert.Equal(t, "./test-prompts", cfg.Agent.PromptBasePath)

	assert.Equal(t, 0.3, cfg.Agent.ReAct.ThinkTemperature)
	assert.Equal(t, 0.4, cfg.Agent.ReAct.ActTemperature)
	assert.Equal(t, 1024, cfg.Agent.ReAct.MaxTokens)
	assert.Equal(t, 60, cfg.Agent.ReAct.Timeout)

	assert.Equal(t, "openai", cfg.Models.DefaultProvider)
	assert.Equal(t, "gpt-4", cfg.Models.DefaultModelName)
	assert.Equal(t, "text-embedding-ada-002", cfg.Models.DefaultEmbedding)

	assert.Equal(t, 9999, cfg.Server.Port)
	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, []string{"http://localhost:3000"}, cfg.Server.CORSOrigins)

	assert.Equal(t, "test-index", cfg.Tools.Memory.DefaultIndex)
	assert.Equal(t, 20, cfg.Tools.Memory.TopK)
	assert.True(t, cfg.Tools.Memory.RerankEnabled)
	assert.Equal(t, "test-rerank", cfg.Tools.Memory.RerankModel)
	assert.Equal(t, 5, cfg.Tools.Memory.RerankTopN)

	assert.Equal(t, "debug", cfg.Logger.Level)
	assert.Equal(t, "text", cfg.Logger.Format)
}

func TestLoadFromFileWithDefaults(t *testing.T) {
	// Test loading a non-existent file returns defaults
	cfg, err := LoadFromFileWithDefaults("/nonexistent/config.yaml")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should have default values
	assert.Equal(t, 10, cfg.Agent.MaxIterations)
	assert.Equal(t, "qwen-max", cfg.Agent.DefaultModel)
	assert.Equal(t, "dashscope", cfg.Models.DefaultProvider)
	assert.Equal(t, 8888, cfg.Server.Port)
}

func TestLoadFromFileMissingFile(t *testing.T) {
	// Test loading a missing file returns error
	cfg, err := LoadFromFile("/nonexistent/config.yaml")
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "configuration file does not exist")
}

func TestLoadFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("AI_AGENT_MAX_ITERATIONS", "25")
	os.Setenv("AI_MODELS_DEFAULT_PROVIDER", "gemini")
	os.Setenv("AI_MODELS_DEFAULT_MODEL_NAME", "gemini-pro")
	os.Setenv("AI_MODELS_DEFAULT_EMBEDDING", "gemini-embedding")
	os.Setenv("AI_SERVER_PORT", "7777")
	os.Setenv("AI_SERVER_HOST", "localhost")

	// Clean up environment variables after test
	defer func() {
		os.Unsetenv("AI_AGENT_MAX_ITERATIONS")
		os.Unsetenv("AI_MODELS_DEFAULT_PROVIDER")
		os.Unsetenv("AI_MODELS_DEFAULT_MODEL_NAME")
		os.Unsetenv("AI_MODELS_DEFAULT_EMBEDDING")
		os.Unsetenv("AI_SERVER_PORT")
		os.Unsetenv("AI_SERVER_HOST")
	}()

	// Load configuration from environment
	cfg, err := LoadFromEnv()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify environment variables override defaults
	assert.Equal(t, 25, cfg.Agent.MaxIterations)
	assert.Equal(t, "gemini", cfg.Models.DefaultProvider)
	assert.Equal(t, "gemini-pro", cfg.Models.DefaultModelName)
	assert.Equal(t, "gemini-embedding", cfg.Models.DefaultEmbedding)
	assert.Equal(t, 7777, cfg.Server.Port)
	assert.Equal(t, "localhost", cfg.Server.Host)
}

func TestSaveToFile(t *testing.T) {
	// Create a test configuration
	cfg := &AIConfig{
		Agent: &AgentConfig{
			MaxIterations:          5,
			StageChannelBufferSize: 3,
			DefaultModel:           "test-model",
			MCPHostName:          "test-host",
			PromptBasePath:       "./test-prompts",
		},
		Models: &ModelsConfig{
			DefaultProvider:   "test-provider",
			DefaultModelName:  "test-model-name",
			DefaultEmbedding:  "test-embedding",
		},
		Server: &ServerConfig{
			Port:        5555,
			Host:        "test-host",
			ReadTimeout: 10,
			WriteTimeout: 10,
		},
		Tools: ToolsConfig{
			Memory: MemoryConfig{
				DefaultIndex: "test-index",
				TopK:         5,
			},
		},
		Logger: &LoggerConfig{
			Level:  "debug",
			Format: "text",
		},
	}

	// Create temporary file path
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "saved-config.yaml")

	// Save configuration to file
	err := SaveToFile(cfg, configPath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(configPath)
	require.NoError(t, err)

	// Load and verify saved configuration
	loadedCfg, err := LoadFromFile(configPath)
	require.NoError(t, err)

	assert.Equal(t, cfg.Agent.MaxIterations, loadedCfg.Agent.MaxIterations)
	assert.Equal(t, cfg.Agent.DefaultModel, loadedCfg.Agent.DefaultModel)
	assert.Equal(t, cfg.Models.DefaultProvider, loadedCfg.Models.DefaultProvider)
	assert.Equal(t, cfg.Server.Port, loadedCfg.Server.Port)
}

func TestFindConfigFile(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	os.Chdir(tempDir)

	// Create a config file
	configPath := filepath.Join(tempDir, "ai-config.yaml")
	err := os.WriteFile(configPath, []byte("agent: {max_iterations: 10}"), 0644)
	require.NoError(t, err)

	// Test finding the config file
	foundPath, err := FindConfigFile()
	require.NoError(t, err)
	assert.Equal(t, configPath, foundPath)
}

func TestFindConfigFileNotFound(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Change to temp directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	os.Chdir(tempDir)

	// Test finding a non-existent config file
	_, err := FindConfigFile()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no configuration file found")
}

func TestGetConfigPaths(t *testing.T) {
	paths := GetConfigPaths()
	assert.Contains(t, paths, "ai-config.yaml")
	assert.Contains(t, paths, "config.yaml")
	assert.Contains(t, paths, "config/ai-config.yaml")
}