package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInjectionContainer(t *testing.T) {
	container := &InjectionContainer{
		providers: make(map[string]interface{}),
	}

	// Test registering a provider
	provider := &configurationAdapter{
		config: &AIConfig{
			Agent: &AgentConfig{
				MaxIterations: 15,
			},
		},
	}

	err := container.RegisterProvider("test", provider)
	require.NoError(t, err)

	// Test registering with empty name
	err = container.RegisterProvider("", provider)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider name cannot be empty")

	// Test registering with nil provider
	err = container.RegisterProvider("nil-test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider cannot be nil")

	// Test retrieving a provider
	retrieved, err := container.GetProvider("test")
	require.NoError(t, err)
	assert.Equal(t, provider, retrieved)

	// Test retrieving non-existent provider
	_, err = container.GetProvider("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider 'nonexistent' not found")
}

func TestGetConfigurationProvider(t *testing.T) {
	// Setup
	originalContainer := container
	container = &InjectionContainer{
		providers: make(map[string]interface{}),
	}
	defer func() {
		container = originalContainer
	}()

	// Register a ConfigurationProvider
	provider := &configurationAdapter{
		config: &AIConfig{
			Agent: &AgentConfig{
				MaxIterations: 20,
			},
		},
	}

	err := RegisterProvider("config", provider)
	require.NoError(t, err)

	// Test getting ConfigurationProvider
	configProvider, err := GetConfigurationProvider("config")
	require.NoError(t, err)
	assert.Equal(t, provider, configProvider)

	// Test getting non-existent provider
	_, err = GetConfigurationProvider("nonexistent")
	assert.Error(t, err)

	// Test getting non-ConfigurationProvider
	err = RegisterProvider("string", "not-a-config-provider")
	require.NoError(t, err)

	_, err = GetConfigurationProvider("string")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not a ConfigurationProvider")
}

func TestGetAgentConfigurationProvider(t *testing.T) {
	// Setup
	originalContainer := container
	container = &InjectionContainer{
		providers: make(map[string]interface{}),
	}
	defer func() {
		container = originalContainer
	}()

	// Register an AgentConfigurationProvider
	provider := &mockAgentProvider{}

	err := RegisterProvider("agent", provider)
	require.NoError(t, err)

	// Test getting AgentConfigurationProvider
	agentProvider, err := GetAgentConfigurationProvider("agent")
	require.NoError(t, err)
	assert.Equal(t, provider, agentProvider)

	// Test getting non-ConfigurationProvider
	err = RegisterProvider("string", "not-a-provider")
	require.NoError(t, err)

	_, err = GetAgentConfigurationProvider("string")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not an AgentConfigurationProvider")
}

func TestGetToolsConfigurationProvider(t *testing.T) {
	// Setup
	originalContainer := container
	container = &InjectionContainer{
		providers: make(map[string]interface{}),
	}
	defer func() {
		container = originalContainer
	}()

	// Register a ToolsConfigurationProvider
	provider := &mockToolsProvider{}

	err := RegisterProvider("tools", provider)
	require.NoError(t, err)

	// Test getting ToolsConfigurationProvider
	toolsProvider, err := GetToolsConfigurationProvider("tools")
	require.NoError(t, err)
	assert.Equal(t, provider, toolsProvider)

	// Test getting non-ConfigurationProvider
	err = RegisterProvider("string", "not-a-provider")
	require.NoError(t, err)

	_, err = GetToolsConfigurationProvider("string")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not a ToolsConfigurationProvider")
}

func TestListProviders(t *testing.T) {
	// Setup
	originalContainer := container
	container = &InjectionContainer{
		providers: make(map[string]interface{}),
	}
	defer func() {
		container = originalContainer
	}()

	// Initially empty
	providers := ListProviders()
	assert.Empty(t, providers)

	// Add some providers
	provider1 := &configurationAdapter{config: &AIConfig{}}
	provider2 := &mockAgentProvider{}

	err := RegisterProvider("config1", provider1)
	require.NoError(t, err)

	err = RegisterProvider("agent", provider2)
	require.NoError(t, err)

	// List providers
	providers = ListProviders()
	assert.Len(t, providers, 2)
	assert.Contains(t, providers, "config1")
	assert.Contains(t, providers, "agent")
}

func TestRemoveProvider(t *testing.T) {
	// Setup
	originalContainer := container
	container = &InjectionContainer{
		providers: make(map[string]interface{}),
	}
	defer func() {
		container = originalContainer
	}()

	// Add a provider
	provider := &configurationAdapter{config: &AIConfig{}}
	err := RegisterProvider("test", provider)
	require.NoError(t, err)

	// Verify it exists
	_, err = GetProvider("test")
	require.NoError(t, err)

	// Remove it
	RemoveProvider("test")

	// Verify it's gone
	_, err = GetProvider("test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider 'test' not found")
}

func TestClearProviders(t *testing.T) {
	// Setup
	originalContainer := container
	container = &InjectionContainer{
		providers: make(map[string]interface{}),
	}
	defer func() {
		container = originalContainer
	}()

	// Add multiple providers
	provider1 := &configurationAdapter{config: &AIConfig{}}
	provider2 := &mockAgentProvider{}

	err := RegisterProvider("config1", provider1)
	require.NoError(t, err)

	err = RegisterProvider("agent", provider2)
	require.NoError(t, err)

	// Verify providers exist
	providers := ListProviders()
	assert.Len(t, providers, 2)

	// Clear all providers
	ClearProviders()

	// Verify all are gone
	providers = ListProviders()
	assert.Empty(t, providers)
}

func TestGetProviderInfo(t *testing.T) {
	// Setup
	originalContainer := container
	container = &InjectionContainer{
		providers: make(map[string]interface{}),
	}
	defer func() {
		container = originalContainer
	}()

	// Add different types of providers
	configProvider := &configurationAdapter{config: &AIConfig{}}
	agentProvider := &mockAgentProvider{}
	toolsProvider := &mockToolsProvider{}
	stringProvider := "just-a-string"

	err := RegisterProvider("config", configProvider)
	require.NoError(t, err)

	err = RegisterProvider("agent", agentProvider)
	require.NoError(t, err)

	err = RegisterProvider("tools", toolsProvider)
	require.NoError(t, err)

	err = RegisterProvider("string", stringProvider)
	require.NoError(t, err)

	// Get provider info
	infos := GetProviderInfo()
	assert.Len(t, infos, 4)

	// Find specific providers
	var configInfo, agentInfo, toolsInfo, stringInfo *ProviderInfo
	for _, info := range infos {
		switch info.Name {
		case "config":
			configInfo = &info
		case "agent":
			agentInfo = &info
		case "tools":
			toolsInfo = &info
		case "string":
			stringInfo = &info
		}
	}

	require.NotNil(t, configInfo)
	assert.Equal(t, "config", configInfo.Name)
	assert.Equal(t, "ConfigurationProvider", configInfo.Type)

	require.NotNil(t, agentInfo)
	assert.Equal(t, "agent", agentInfo.Name)
	assert.Equal(t, "*config.mockAgentProvider", agentInfo.Type)

	require.NotNil(t, toolsInfo)
	assert.Equal(t, "tools", toolsInfo.Name)
	assert.Equal(t, "*config.mockToolsProvider", toolsInfo.Type)

	require.NotNil(t, stringInfo)
	assert.Equal(t, "string", stringInfo.Name)
	assert.Equal(t, "string", stringInfo.Type)
}

// Mock implementations for testing

type mockAgentProvider struct {
	AgentConfigurationProvider
}

func (m *mockAgentProvider) GetMaxIterations() int {
	return 42
}

func (m *mockAgentProvider) GetTemperature(phase string) float64 {
	return 0.7
}

func (m *mockAgentProvider) GetStageChannelBuffer() int {
	return 15
}

func (m *mockAgentProvider) GetPromptDir() string {
	return "/mock/prompts"
}

type mockToolsProvider struct {
	ToolsConfigurationProvider
}

func (m *mockToolsProvider) GetMCPHost() string {
	return "mock-mcp-host"
}

func (m *mockToolsProvider) GetMemoryConfig() *MemoryConfig {
	return &MemoryConfig{
		DefaultIndex: "mock-index",
		TopK:         25,
	}
}