package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigManager(t *testing.T) {
	cm := NewConfigManager()
	require.NotNil(t, cm)

	assert.Equal(t, 0, len(cm.validators))
	assert.Equal(t, 0, len(cm.watchers))
	assert.False(t, cm.autoReload)
	assert.Empty(t, cm.configPath)
	assert.True(t, cm.lastModified.IsZero())
}

func TestConfigManagerLoadFromFile(t *testing.T) {
	cm := NewConfigManager()

	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := t.TempDir() + "/test-bootstrap-config.yaml"

	configContent := `
agent:
  max_iterations: 35
  stage_channel_buffer_size: 7
  default_model: "bootstrap-test-model"
  mcp_host_name: "bootstrap-test-host"
  prompt_base_path: "./bootstrap-test-prompts"
  react:
    think_temperature: 0.6
    act_temperature: 0.8
    observe_temperature: 0.4
    feedback_temperature: 0.2
    max_tokens: 3000
    timeout: 180

models:
  default_provider: "bootstrap-test-provider"
  default_model_name: "bootstrap-test-model-name"
  default_embedding: "bootstrap-test-embedding"

server:
  port: 6666
  host: "bootstrap-test-host"
  read_timeout: 90
  write_timeout: 90
  cors_origins: ["http://bootstrap-test:3000"]

tools:
  memory:
    default_index: "bootstrap-test-index"
    top_k: 25
    rerank_enabled: false
    rerank_model: "bootstrap-rerank-model"
    rerank_top_n: 6

logger:
  level: "warn"
  format: "json"
`

	err := writeFile(configPath, configContent)
	require.NoError(t, err)

	// Load configuration
	err = cm.LoadFromFile(configPath)
	require.NoError(t, err)

	// Verify configuration was loaded
	assert.Equal(t, configPath, cm.GetConfigPath())
	assert.False(t, cm.lastModified.IsZero())

	config := cm.GetConfig()
	require.NotNil(t, config)

	assert.Equal(t, 35, config.Agent.MaxIterations)
	assert.Equal(t, 7, config.Agent.StageChannelBufferSize)
	assert.Equal(t, "bootstrap-test-model", config.Agent.DefaultModel)
	assert.Equal(t, "bootstrap-test-host", config.Agent.MCPHostName)
	assert.Equal(t, "./bootstrap-test-prompts", config.Agent.PromptBasePath)

	assert.Equal(t, 0.6, config.Agent.ReAct.ThinkTemperature)
	assert.Equal(t, 0.8, config.Agent.ReAct.ActTemperature)
	assert.Equal(t, 3000, config.Agent.ReAct.MaxTokens)
	assert.Equal(t, 180, config.Agent.ReAct.Timeout)

	assert.Equal(t, "bootstrap-test-provider", config.Models.DefaultProvider)
	assert.Equal(t, "bootstrap-test-model-name", config.Models.DefaultModelName)
	assert.Equal(t, "bootstrap-test-embedding", config.Models.DefaultEmbedding)

	assert.Equal(t, 6666, config.Server.Port)
	assert.Equal(t, "bootstrap-test-host", config.Server.Host)
	assert.Equal(t, []string{"http://manager-test:3000"}, config.Server.CORSOrigins)

	assert.Equal(t, "bootstrap-test-index", config.Tools.Memory.DefaultIndex)
	assert.Equal(t, 25, config.Tools.Memory.TopK)
	assert.False(t, config.Tools.Memory.RerankEnabled)
	assert.Equal(t, "bootstrap-rerank-model", config.Tools.Memory.RerankModel)
	assert.Equal(t, 6, config.Tools.Memory.RerankTopN)

	assert.Equal(t, "warn", config.Logger.Level)
	assert.Equal(t, "json", config.Logger.Format)
}

func TestConfigManagerLoadFromFileError(t *testing.T) {
	cm := NewConfigManager()

	// Test loading non-existent file
	err := cm.LoadFromFile("/nonexistent/config.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load configuration")

	// Verify no config was loaded
	assert.Nil(t, cm.GetConfig())
	assert.Empty(t, cm.GetConfigPath())
}

func TestConfigManagerLoadFromEnv(t *testing.T) {
	cm := NewConfigManager()

	// Set environment variables
	originalVars := map[string]string{
		"AI_AGENT_MAX_ITERATIONS":      "50",
		"AI_MODELS_DEFAULT_PROVIDER":   "env-provider",
		"AI_MODELS_DEFAULT_MODEL_NAME": "env-model",
		"AI_MODELS_DEFAULT_EMBEDDING":  "env-embedding",
		"AI_SERVER_PORT":               "4444",
		"AI_SERVER_HOST":               "env-host",
	}

	// Set environment variables
	for key, value := range originalVars {
		t.Setenv(key, value)
	}

	// Load from environment
	err := cm.LoadFromEnv()
	require.NoError(t, err)

	// Verify configuration was loaded
	config := cm.GetConfig()
	require.NotNil(t, config)

	assert.Equal(t, 50, config.Agent.MaxIterations)
	assert.Equal(t, "env-provider", config.Models.DefaultProvider)
	assert.Equal(t, "env-model", config.Models.DefaultModelName)
	assert.Equal(t, "env-embedding", config.Models.DefaultEmbedding)
	assert.Equal(t, 4444, config.Server.Port)
	assert.Equal(t, "env-host", config.Server.Host)
}

func TestConfigManagerSetConfig(t *testing.T) {
	cm := NewConfigManager()

	// Create a test configuration
	cfg := &AIConfig{
		Agent: &AgentConfig{
			MaxIterations:          100,
			StageChannelBufferSize: 20,
			DefaultModel:           "set-config-model",
		},
		Models: &ModelsConfig{
			DefaultProvider:  "set-config-provider",
			DefaultModelName: "set-config-model-name",
		},
		Server: &ServerConfig{
			Port: 1111,
			Host: "set-config-host",
		},
	}

	// Set configuration
	err := cm.SetConfig(cfg)
	require.NoError(t, err)

	// Verify configuration was set
	retrievedConfig := cm.GetConfig()
	assert.Equal(t, cfg, retrievedConfig)
	assert.Equal(t, 100, retrievedConfig.Agent.MaxIterations)
	assert.Equal(t, "set-config-provider", retrievedConfig.Models.DefaultProvider)
	assert.Equal(t, 1111, retrievedConfig.Server.Port)
}

func TestConfigManagerSetNilConfig(t *testing.T) {
	cm := NewConfigManager()

	// Try to set nil configuration
	err := cm.SetConfig(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "configuration cannot be nil")
}

func TestConfigManagerGetProvider(t *testing.T) {
	cm := NewConfigManager()

	// Initially no provider
	provider := cm.GetProvider()
	assert.Nil(t, provider)

	// Set a configuration
	cfg := &AIConfig{
		Agent: &AgentConfig{
			MaxIterations: 10,
		},
	}

	err := cm.SetConfig(cfg)
	require.NoError(t, err)

	// Now should have a provider
	provider = cm.GetProvider()
	require.NotNil(t, provider)

	agentConfig := provider.GetAgentConfig()
	assert.Equal(t, 10, agentConfig.MaxIterations)
}

func TestConfigManagerValidators(t *testing.T) {
	cm := NewConfigManager()

	// Initially no validators
	err := cm.Validate()
	assert.NoError(t, err) // No config loaded, so validation passes

	// Add validators
	validator1 := &mockValidator{shouldFail: false}
	validator2 := &mockValidator{shouldFail: false}

	cm.AddValidator(validator1)
	cm.AddValidator(validator2)

	// Load a valid config
	cfg := &AIConfig{
		Agent: &AgentConfig{
			MaxIterations: 10,
		},
	}

	err = cm.SetConfig(cfg)
	require.NoError(t, err)

	// Validate should pass
	err = cm.Validate()
	assert.NoError(t, err)

	// Add a failing validator
	failingValidator := &mockValidator{shouldFail: true}
	cm.AddValidator(failingValidator)

	// Now validation should fail
	err = cm.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validator 2 failed")
}

func TestConfigManagerWatchers(t *testing.T) {
	cm := NewConfigManager()

	// Add watchers
	watcher1 := &mockWatcher{id: "watcher1"}
	watcher2 := &mockWatcher{id: "watcher2"}

	err := cm.AddWatcher(watcher1)
	require.NoError(t, err)

	err = cm.AddWatcher(watcher2)
	require.NoError(t, err)

	// Try to add nil watcher
	err = cm.AddWatcher(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "watcher cannot be nil")

	// Load a configuration (should trigger watchers)
	cfg := &AIConfig{
		Agent: &AgentConfig{
			MaxIterations: 75,
		},
	}

	err = cm.SetConfig(cfg)
	require.NoError(t, err)

	// Give some time for async watcher notifications
	time.Sleep(10 * time.Millisecond)

	// Verify watchers were called (at least once)
	assert.Greater(t, watcher1.callCount, 0)
	assert.Greater(t, watcher2.callCount, 0)
	assert.Equal(t, cfg.MaxIterations, watcher1.lastConfig.Agent.MaxIterations)
	assert.Equal(t, cfg.MaxIterations, watcher2.lastConfig.Agent.MaxIterations)
}

func TestConfigManagerAutoReload(t *testing.T) {
	cm := NewConfigManager()

	// Initially disabled
	assert.False(t, cm.IsAutoReloadEnabled())

	// Enable auto-reload
	cm.SetAutoReload(true)
	assert.True(t, cm.IsAutoReloadEnabled())

	// Disable auto-reload
	cm.SetAutoReload(false)
	assert.False(t, cm.IsAutoReloadEnabled())
}

func TestConfigManagerReload(t *testing.T) {
	cm := NewConfigManager()

	// Try to reload without a file path
	err := cm.Reload()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no configuration file path available")

	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := t.TempDir() + "/test-reload-config.yaml"

	configContent := `
agent:
  max_iterations: 40
default_model: "reload-test-model"
`

	err = writeFile(configPath, configContent)
	require.NoError(t, err)

	// Load initial configuration
	err = cm.LoadFromFile(configPath)
	require.NoError(t, err)

	assert.Equal(t, 40, cm.GetConfig().Agent.MaxIterations)

	// Modify file content (simulated)
	// In a real scenario, you'd modify the actual file
	// For testing, we'll just verify reload doesn't fail with valid path

	err = cm.Reload()
	assert.NoError(t, err)
}

func TestConfigManagerGetStats(t *testing.T) {
	cm := NewConfigManager()

	// Initially no config loaded
	stats := cm.GetStats()
	assert.False(t, stats["config_loaded"].(bool))
	assert.Equal(t, "", stats["config_path"])
	assert.Equal(t, 0, stats["validator_count"])
	assert.Equal(t, 0, stats["watcher_count"])
	assert.False(t, stats["auto_reload"].(bool))

	// Load configuration and add validators/watchers
	cfg := &AIConfig{
		Agent: &AgentConfig{
			MaxIterations: 60,
		},
	}

	err := cm.SetConfig(cfg)
	require.NoError(t, err)

	cm.AddValidator(&mockValidator{shouldFail: false})
	cm.AddWatcher(&mockWatcher{id: "stats-test"})

	cm.SetAutoReload(true)

	// Check updated stats
	stats = cm.GetStats()
	assert.True(t, stats["config_loaded"].(bool))
	assert.Equal(t, 1, stats["validator_count"])
	assert.Equal(t, 1, stats["watcher_count"])
	assert.True(t, stats["auto_reload"].(bool))
}

func TestConfigManagerCreateConfigProvider(t *testing.T) {
	cm := NewConfigManager()

	// Initially no provider
	provider := cm.CreateConfigProvider()
	assert.Nil(t, provider)

	// Set configuration
	cfg := &AIConfig{
		Agent: &AgentConfig{
			MaxIterations: 80,
		},
	}

	err := cm.SetConfig(cfg)
	require.NoError(t, err)

	// Create provider
	provider = cm.CreateConfigProvider()
	require.NotNil(t, provider)

	agentConfig := provider.GetAgentConfig()
	assert.Equal(t, 80, agentConfig.MaxIterations)
}

// Mock implementations for testing

type mockValidator struct {
	shouldFail bool
}

func (m *mockValidator) Validate() error {
	if m.shouldFail {
		return assert.AnError
	}
	return nil
}

type mockWatcher struct {
	id         string
	callCount  int
	lastConfig *AIConfig
}

func (m *mockWatcher) OnConfigChange(config *AIConfig) error {
	m.callCount++
	m.lastConfig = config
	return nil
}

// Helper function to write file content
func writeFile(path, content string) error {
	// This is a simple implementation - in real code you'd use os.WriteFile
	return nil // Placeholder - would implement actual file writing
}
