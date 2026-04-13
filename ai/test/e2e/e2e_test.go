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

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"dubbo-admin-ai/component/agent/react"
	"dubbo-admin-ai/component/logger"
	"dubbo-admin-ai/component/memory"
	"dubbo-admin-ai/component/models"
	compRag "dubbo-admin-ai/component/rag"
	"dubbo-admin-ai/component/server"
	"dubbo-admin-ai/component/tools"
	appruntime "dubbo-admin-ai/runtime"
)

func init() {
	// Set schema directory relative to test file
	_, file, _, ok := runtime.Caller(0)
	if ok {
		// ai/test/e2e/e2e_test.go -> need to go up 3 levels to get ai directory
		aiDir := filepath.Dir(filepath.Dir(filepath.Dir(file)))
		schemaDir := filepath.Join(aiDir, "schema", "json")
		if abs, err := filepath.Abs(schemaDir); err == nil {
			_ = os.Setenv("SCHEMA_DIR", abs)
		}

		// Load .env file from ai directory for API keys
		envFile := filepath.Join(aiDir, ".env")
		if err := loadDotEnvHelper(envFile); err != nil {
			// .env not found is acceptable, only log real errors
			errMsg := err.Error()
			if !strings.Contains(errMsg, "no such file") &&
				!strings.Contains(errMsg, "file does not exist") &&
				!strings.Contains(errMsg, "cannot find the file") {
				// Can't log in init(), just ignore
			}
		}
	}
}

// TestServerE2E tests the full server startup and API endpoints
func TestServerE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	// Setup test config
	ctx := context.Background()
	configPath, _ := createTestConfig(t)
	rt, err := appruntime.Bootstrap(configPath, registerFactories)
	if err != nil {
		t.Fatalf("Failed to bootstrap runtime: %v", err)
	}
	defer rt.StopAll()

	// Get server component
	serverComp, err := rt.GetComponent("server")
	if err != nil {
		t.Fatalf("Failed to get server component: %v", err)
	}

	svr, ok := serverComp.(*server.ServerComponent)
	if !ok {
		t.Fatalf("Server component is not the expected type")
	}

	// Start server in background
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- svr.Start()
	}()

	// Wait for server to be ready
	time.Sleep(2 * time.Second)

	// Get server address
	baseURL := fmt.Sprintf("http://0.0.0.0:8880")

	t.Run("health_check", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/health", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Health check status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
	})

	t.Run("create_session", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/v1/ai/sessions", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Create session failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Create session status = %d, body: %s", resp.StatusCode, string(body))
		}

		var result struct {
			Message string `json:"message"`
			Data    struct {
				SessionID string `json:"session_id"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result.Data.SessionID == "" {
			t.Fatalf("Session ID is empty")
		}

		t.Logf("Created session: %s", result.Data.SessionID)
	})

	t.Run("list_sessions", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/v1/ai/sessions", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("List sessions failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("List sessions status = %d, body: %s", resp.StatusCode, string(body))
		}

		var result struct {
			Message string `json:"message"`
			Data    struct {
				Sessions []struct {
					SessionID string `json:"session_id"`
				} `json:"sessions"`
				Total int `json:"total"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if result.Data.Total < 1 {
			t.Logf("Warning: No sessions found (expected at least 1)")
		}
	})

	// Check if server started successfully
	select {
	case err := <-serverErr:
		if err != nil {
			t.Logf("Server error (expected during shutdown): %v", err)
		}
	default:
	}
}

// TestComponentIntegration tests component integration without HTTP server
func TestComponentIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	t.Run("bootstrap_all_components", func(t *testing.T) {
		configPath, _ := createTestConfig(t)
		rt, err := appruntime.Bootstrap(configPath, registerFactories)
		if err != nil {
			t.Fatalf("Failed to bootstrap runtime: %v", err)
		}
		defer rt.StopAll()

		// Verify all components are registered
		components := rt.ComponentSnapshot()
		t.Logf("Components loaded: %s", formatComponents(components))

		// Check critical components
		criticalTypes := []string{"logger", "memory", "models", "rag", "tools", "server", "agent"}
		for _, compType := range criticalTypes {
			comps, err := rt.GetComponentByType(compType)
			if err != nil {
				t.Fatalf("Component type %s not found: %v", compType, err)
			}
			if len(comps) == 0 {
				t.Fatalf("No components of type %s found", compType)
			}
		}
	})

	t.Run("rag_component_initialization", func(t *testing.T) {
		configPath, _ := createTestConfig(t)
		rt, err := appruntime.Bootstrap(configPath, registerFactories)
		if err != nil {
			t.Fatalf("Failed to bootstrap runtime: %v", err)
		}
		defer rt.StopAll()

		ragComp, err := rt.GetComponent("rag")
		if err != nil {
			t.Fatalf("Failed to get RAG component: %v", err)
		}

		if ragComp.Name() == "" {
			t.Fatalf("RAG component has no name")
		}

		t.Logf("RAG component initialized: %s", ragComp.Name())
	})

	t.Run("agent_component_initialization", func(t *testing.T) {
		configPath, _ := createTestConfig(t)
		rt, err := appruntime.Bootstrap(configPath, registerFactories)
		if err != nil {
			t.Fatalf("Failed to bootstrap runtime: %v", err)
		}
		defer rt.StopAll()

		agentComp, err := rt.GetComponent("agent")
		if err != nil {
			t.Fatalf("Failed to get agent component: %v", err)
		}

		if agentComp.Name() == "" {
			t.Fatalf("Agent component has no name")
		}

		t.Logf("Agent component initialized: %s", agentComp.Name())
	})
}

// TestAPIWithMockSession tests API with mock session
func TestAPIWithMockSession(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	ctx := context.Background()
	configPath, _ := createTestConfig(t)
	rt, err := appruntime.Bootstrap(configPath, registerFactories)
	if err != nil {
		t.Fatalf("Failed to bootstrap runtime: %v", err)
	}
	defer rt.StopAll()

	serverComp, err := rt.GetComponent("server")
	if err != nil {
		t.Fatalf("Failed to get server component: %v", err)
	}

	svr, ok := serverComp.(*server.ServerComponent)
	if !ok {
		t.Fatalf("Server component is not the expected type")
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- svr.Start()
	}()
	defer func() {
		// Stop server
		svr.Stop()
	}()

	time.Sleep(2 * time.Second)
	baseURL := fmt.Sprintf("http://0.0.0.0:8880")

	t.Run("get_mock_session", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/v1/ai/sessions/session_test", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Get session failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Get session status = %d, body: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("chat_with_mock_session", func(t *testing.T) {
		chatReq := map[string]string{
			"message":   "Hello, who are you?",
			"sessionID": "session_test",
		}
		body, err := json.Marshal(chatReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/v1/ai/chat/stream", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Chat request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("Chat status = %d, body: %s", resp.StatusCode, string(body))
		}

		// Read SSE stream
		buf := new(strings.Builder)
		chunk := make([]byte, 1024)
		for {
			n, err := resp.Body.Read(chunk)
			if n > 0 {
				buf.Write(chunk[:n])
			}
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Logf("Read error (non-fatal): %v", err)
				break
			}
			// Stop after reading some data
			if buf.Len() > 1000 {
				break
			}
		}

		response := buf.String()
		if !strings.Contains(response, "event:") && !strings.Contains(response, "data:") {
			t.Logf("Warning: No SSE events found, response: %s", response[:min(200, len(response))])
		}
	})
}

func TestConfigurationLoading(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	t.Run("load_all_configs", func(t *testing.T) {
		_, file, _, _ := runtime.Caller(0)
		aiDir := filepath.Dir(filepath.Dir(filepath.Dir(file)))

		// Helper to convert path to forward slashes for YAML
		toSlash := func(p string) string {
			return strings.ReplaceAll(p, "\\", "/")
		}

		// Create temporary config with absolute paths
		tmpDir := t.TempDir()
		configContent := fmt.Sprintf(`project: dubbo-admin-ai
version: 1.0.0
components:
  logger: %s
  memory: %s
  models: %s
  server: %s
  tools: %s
  rag: %s
  agent: %s
`,
			toSlash(filepath.Join(aiDir, "component", "logger", "logger.yaml")),
			toSlash(filepath.Join(aiDir, "component", "memory", "memory.yaml")),
			toSlash(filepath.Join(aiDir, "component", "models", "models.yaml")),
			toSlash(filepath.Join(aiDir, "component", "server", "server.yaml")),
			toSlash(filepath.Join(aiDir, "component", "tools", "tools.yaml")),
			toSlash(filepath.Join(aiDir, "component", "rag", "rag.yaml")),
			toSlash(filepath.Join(aiDir, "component", "agent", "agent.yaml")),
		)

		// Create a modified agent.yaml with absolute prompt path
		agentConfigPath := filepath.Join(tmpDir, "agent.yaml")
		agentContent, _ := os.ReadFile(filepath.Join(aiDir, "component", "agent", "agent.yaml"))
		modifiedAgentContent := string(agentContent)
		modifiedAgentContent = strings.Replace(modifiedAgentContent, `prompt_base_path: "./prompts"`,
			fmt.Sprintf(`prompt_base_path: "%s"`, toSlash(filepath.Join(aiDir, "prompts"))), 1)
		_ = os.WriteFile(agentConfigPath, []byte(modifiedAgentContent), 0644)

		// Update config to use our modified agent.yaml
		configContent = strings.Replace(configContent,
			toSlash(filepath.Join(aiDir, "component", "agent", "agent.yaml")),
			toSlash(agentConfigPath), 1)

		configPath := filepath.Join(tmpDir, "config.yaml")
		_ = os.WriteFile(configPath, []byte(configContent), 0644)

		rt, err := appruntime.Bootstrap(configPath, registerFactories)
		if err != nil {
			t.Fatalf("Failed to bootstrap runtime: %v", err)
		}
		defer rt.StopAll()

		// Test configuration validation
		_, err = rt.GetComponentByType("logger")
		if err != nil {
			t.Fatalf("Logger config failed: %v", err)
		}

		_, err = rt.GetComponentByType("models")
		if err != nil {
			t.Fatalf("Models config failed: %v", err)
		}

		_, err = rt.GetComponentByType("rag")
		if err != nil {
			t.Fatalf("RAG config failed: %v", err)
		}

		_, err = rt.GetComponentByType("agent")
		if err != nil {
			t.Fatalf("Agent config failed: %v", err)
		}
	})
}

// Helper functions

func registerFactories(rt *appruntime.Runtime) {
	rt.RegisterFactory("logger", logger.LoggerFactory)
	rt.RegisterFactory("memory", memory.MemoryFactory)
	rt.RegisterFactory("models", models.ModelsFactory)
	rt.RegisterFactory("rag", compRag.RAGFactory)
	rt.RegisterFactory("tools", tools.ToolsFactory)
	rt.RegisterFactory("server", server.ServerFactory)
	rt.RegisterFactory("agent", react.AgentFactory)
}

// createTestConfig creates a temporary config with absolute paths for testing
func createTestConfig(t *testing.T) (configPath string, cleanup func()) {
	_, file, _, _ := runtime.Caller(0)
	aiDir := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	tmpDir := t.TempDir()

	// Helper to convert path to forward slashes for YAML
	toSlash := func(p string) string {
		return strings.ReplaceAll(p, "\\", "/")
	}

	// Create a modified agent.yaml with absolute prompt path
	agentConfigPath := filepath.Join(tmpDir, "agent.yaml")
	agentContent, _ := os.ReadFile(filepath.Join(aiDir, "component", "agent", "agent.yaml"))
	modifiedAgentContent := string(agentContent)
	modifiedAgentContent = strings.Replace(modifiedAgentContent, `prompt_base_path: "./prompts"`,
		fmt.Sprintf(`prompt_base_path: "%s"`, toSlash(filepath.Join(aiDir, "prompts"))), 1)
	_ = os.WriteFile(agentConfigPath, []byte(modifiedAgentContent), 0644)

	// Create config file with absolute paths
	configContent := fmt.Sprintf(`project: dubbo-admin-ai
version: 1.0.0
components:
  logger: %s
  memory: %s
  models: %s
  server: %s
  tools: %s
  rag: %s
  agent: %s
`,
		toSlash(filepath.Join(aiDir, "component", "logger", "logger.yaml")),
		toSlash(filepath.Join(aiDir, "component", "memory", "memory.yaml")),
		toSlash(filepath.Join(aiDir, "component", "models", "models.yaml")),
		toSlash(filepath.Join(aiDir, "component", "server", "server.yaml")),
		toSlash(filepath.Join(aiDir, "component", "tools", "tools.yaml")),
		toSlash(filepath.Join(aiDir, "component", "rag", "rag.yaml")),
		toSlash(agentConfigPath),
	)

	configPath = filepath.Join(tmpDir, "config.yaml")
	_ = os.WriteFile(configPath, []byte(configContent), 0644)

	cleanup = func() {} // tmpDir will be cleaned up by t.TempDir()
	return configPath, cleanup
}

func formatComponents(snapshot []appruntime.ComponentSnapshotEntry) string {
	var sb strings.Builder
	for _, entry := range snapshot {
		sb.WriteString(fmt.Sprintf("%s(%d), ", entry.Type, entry.Count))
	}
	return sb.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// loadDotEnvHelper loads .env file from specified path
func loadDotEnvHelper(envFile string) error {
	// godotenv.Load() loads from current directory, so we need to
	// change directory, load, then restore
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to the directory containing .env
	envDir := filepath.Dir(envFile)
	if err := os.Chdir(envDir); err != nil {
		return err
	}

	// Load .env (now .env is in current directory)
	if err := godotenv.Load(); err != nil {
		return err
	}

	return nil
}
