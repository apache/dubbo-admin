package config_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"dubbo-admin-ai/component/agent/react"
	"dubbo-admin-ai/component/server"
	"dubbo-admin-ai/config"
)

func repoSchemaDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get caller")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	return filepath.Join(root, "schema", "json")
}

func copySchemaDir(t *testing.T, dst string) {
	t.Helper()
	src := repoSchemaDir(t)
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("read schema dir: %v", err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatalf("mkdir schema dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatalf("read schema file %s: %v", e.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), b, 0o644); err != nil {
			t.Fatalf("write schema file %s: %v", e.Name(), err)
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustContain(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got: %v", want, err)
	}
}

func TestLoader_MainConfig_Parse(t *testing.T) {
	tests := []struct {
		name       string
		mainYAML   string
		expectLike string
	}{
		{name: "parse_error", mainYAML: "project: p\nversion: v\ncomponents: [", expectLike: "parse error"},
		{name: "parse_error_priority", mainYAML: "project: p\nversion v\ncomponents:\n  logger: logger.yaml\nunknown: true\n", expectLike: "parse error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("SCHEMA_DIR", repoSchemaDir(t))
			writeFile(t, filepath.Join(dir, "config.yaml"), tt.mainYAML)
			loader := config.NewLoader(filepath.Join(dir, "config.yaml"))
			_, err := loader.Load()
			mustContain(t, err, tt.expectLike)
		})
	}
}

func TestLoader_Component_Parse(t *testing.T) {
	tests := []struct {
		name         string
		componentYML string
	}{
		{name: "parse_error", componentYML: "type: logger\nspec: ["},
		{name: "parse_error_priority", componentYML: "type: logger\nspec: [\nextra: true\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("SCHEMA_DIR", repoSchemaDir(t))
			writeFile(t, filepath.Join(dir, "config.yaml"), "project: p\nversion: v\ncomponents:\n  logger: logger.yaml\n")
			writeFile(t, filepath.Join(dir, "logger.yaml"), tt.componentYML)
			loader := config.NewLoader(filepath.Join(dir, "config.yaml"))
			_, err := loader.Load()
			mustContain(t, err, "parse error")
		})
	}
}

func TestLoader_MainConfig_Structural(t *testing.T) {
	tests := []struct {
		name       string
		mainYAML   string
		expectLike string
	}{
		{name: "unknown_field", mainYAML: "project: p\nversion: v\nunknown: true\ncomponents:\n  logger: logger.yaml\n", expectLike: "structural error"},
		{name: "components_type_invalid", mainYAML: "project: p\nversion: v\ncomponents:\n  logger:\n    path: logger.yaml\n", expectLike: "structural error"},
		{name: "components_array_item_invalid", mainYAML: "project: p\nversion: v\ncomponents:\n  agent:\n    - a.yaml\n    - 1\n", expectLike: "structural error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("SCHEMA_DIR", repoSchemaDir(t))
			writeFile(t, filepath.Join(dir, "config.yaml"), tt.mainYAML)
			loader := config.NewLoader(filepath.Join(dir, "config.yaml"))
			_, err := loader.Load()
			mustContain(t, err, tt.expectLike)
		})
	}
}

func TestLoader_MainConfig_SchemaDir(t *testing.T) {
	dir := t.TempDir()
	schemaDst := filepath.Join(dir, "schema", "json")
	copySchemaDir(t, schemaDst)
	t.Setenv("SCHEMA_DIR", "")

	writeFile(t, filepath.Join(dir, "config.yaml"), "project: p\nversion: v\ncomponents:\n  logger: logger.yaml\n")
	writeFile(t, filepath.Join(dir, "logger.yaml"), "type: logger\nspec: {}\n")

	loader := config.NewLoader(filepath.Join(dir, "config.yaml"))
	loaded, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	want, _ := filepath.Abs(schemaDst)
	if loaded.SchemaDir != want {
		t.Fatalf("SchemaDir = %q, want %q", loaded.SchemaDir, want)
	}
	loggerConfigs := loaded.Components["logger"]
	if len(loggerConfigs) != 1 {
		t.Fatalf("logger configs len = %d, want 1", len(loggerConfigs))
	}
	if loggerConfigs[0].Type != "logger" {
		t.Fatalf("logger config type = %q, want logger", loggerConfigs[0].Type)
	}
}

func TestLoader_Component_Structural(t *testing.T) {
	tests := []struct {
		name         string
		componentYML string
	}{
		{name: "missing_type", componentYML: "spec: {}\n"},
		{name: "missing_spec", componentYML: "type: server\n"},
		{name: "unknown_top_field", componentYML: "type: server\nspec: {}\nextra: true\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("SCHEMA_DIR", repoSchemaDir(t))
			writeFile(t, filepath.Join(dir, "comp.yaml"), tt.componentYML)
			loader := config.NewLoader(filepath.Join(dir, "config.yaml"))
			_, err := loader.LoadComponent("comp.yaml")
			mustContain(t, err, "structural error")
		})
	}
}

func TestLoader_Component_DefaultInjection(t *testing.T) {
	tests := []struct {
		name         string
		fileName     string
		componentYML string
		assertFn     func(t *testing.T, cfg *config.Config)
	}{
		{
			name:         "server",
			fileName:     "server.yaml",
			componentYML: "type: server\nspec: {}\n",
			assertFn: func(t *testing.T, cfg *config.Config) {
				var spec server.ServerSpec
				if err := cfg.Spec.Decode(&spec); err != nil {
					t.Fatalf("decode server spec: %v", err)
				}
				if spec.Port != 8888 || spec.Host != "0.0.0.0" || spec.ReadTimeout != 30 || spec.WriteTimeout != 30 {
					t.Fatalf("server defaults not injected: %+v", spec)
				}
			},
		},
		{
			name:     "rag_splitter_oneof",
			fileName: "rag.yaml",
			componentYML: `type: rag
spec:
  embedder:
    spec:
      model: dashscope/qwen3-embedding
  loader:
    spec: {}
  splitter:
    spec: {}
  indexer:
    spec: {}
  retriever:
    spec: {}
`,
			assertFn: func(t *testing.T, cfg *config.Config) {
				var spec map[string]any
				if err := cfg.Spec.Decode(&spec); err != nil {
					t.Fatalf("decode rag spec: %v", err)
				}
				splitter, ok := spec["splitter"].(map[string]any)
				if !ok {
					t.Fatalf("splitter = %T, want map[string]any", spec["splitter"])
				}
				if splitter["type"] != "recursive" {
					t.Fatalf("splitter.type = %v, want recursive", splitter["type"])
				}
				splitterSpec, ok := splitter["spec"].(map[string]any)
				if !ok {
					t.Fatalf("splitter.spec = %T, want map[string]any", splitter["spec"])
				}
				if splitterSpec["chunk_size"] != 1000 || splitterSpec["overlap_size"] != 100 {
					t.Fatalf("splitter defaults not injected: %+v", splitterSpec)
				}
				if _, exists := splitterSpec["headers"]; exists {
					t.Fatalf("unexpected markdown defaults injected: %+v", splitterSpec)
				}
			},
		},
		{
			name:     "agent",
			fileName: "agent.yaml",
			componentYML: `type: agent
spec:
  model: qwen-max
  prompt_base_path: ./prompts
  stages:
    - name: reasonAct
      flow_type: reasonAct
      prompt_file: agentReasonAct.txt
`,
			assertFn: func(t *testing.T, cfg *config.Config) {
				var spec react.AgentSpec
				if err := cfg.Spec.Decode(&spec); err != nil {
					t.Fatalf("decode agent spec: %v", err)
				}
				if len(spec.Stages) != 1 {
					t.Fatalf("stages len = %d, want 1", len(spec.Stages))
				}
				stage := spec.Stages[0]
				if stage.Temperature == 0 || stage.TopP == 0 || stage.MaxTokens == 0 || stage.Timeout == 0 {
					t.Fatalf("agent stage defaults not injected: %+v", stage)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("SCHEMA_DIR", repoSchemaDir(t))
			writeFile(t, filepath.Join(dir, tt.fileName), tt.componentYML)
			loader := config.NewLoader(filepath.Join(dir, "config.yaml"))
			cfg, err := loader.LoadComponent(tt.fileName)
			if err != nil {
				t.Fatalf("LoadComponent() error: %v", err)
			}
			tt.assertFn(t, cfg)
		})
	}
}

func TestLoader_Component_ConditionalRequired(t *testing.T) {
	tests := []struct {
		name         string
		fileName     string
		componentYML string
	}{
		{name: "tools_mcp_enabled_require_host", fileName: "tools.yaml", componentYML: `type: tools
spec:
  enable_mcp_tools: true
  mcp_host_name: ""
`},
		{name: "rag_reranker_enabled_require_api_key", fileName: "rag.yaml", componentYML: `type: rag
spec:
  embedder:
    spec:
      model: dashscope/qwen3-embedding
  loader:
    spec: {}
  splitter:
    spec: {}
  indexer:
    spec: {}
  retriever:
    spec: {}
  reranker:
    spec:
      enabled: true
`},
		{name: "rag_splitter_oneof_branch_validation", fileName: "rag.yaml", componentYML: `type: rag
spec:
  embedder:
    spec:
      model: dashscope/qwen3-embedding
  loader:
    spec: {}
  splitter:
    type: recursive
    spec:
      chunk_size: 10
      headers:
        "#": h1
  indexer:
    spec: {}
  retriever:
    spec: {}
`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("SCHEMA_DIR", repoSchemaDir(t))
			writeFile(t, filepath.Join(dir, tt.fileName), tt.componentYML)
			loader := config.NewLoader(filepath.Join(dir, "config.yaml"))
			_, err := loader.LoadComponent(tt.fileName)
			mustContain(t, err, "structural error")
		})
	}
}
