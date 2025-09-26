package config

import (
	"path/filepath"
	"runtime"

	"dubbo-admin-ai/plugins/dashscope"
	"dubbo-admin-ai/plugins/model"
)

var (
	// API keys
	GEMINI_API_KEY      string
	SILICONFLOW_API_KEY string
	DASHSCOPE_API_KEY   string
	PINECONE_API_KEY    string
	COHERE_API_KEY      string
	PROMETHEUS_URL      string

	// Configuration
	// Automatically get project root directory
	_, b, _, _   = runtime.Caller(0)
	PROJECT_ROOT = filepath.Join(filepath.Dir(b), "..")

	PROMPT_DIR_PATH string          = filepath.Join(PROJECT_ROOT, "prompts")
	DEFAULT_MODEL   *model.Model    = dashscope.Qwen_max
	EMBEDDING_MODEL *model.Embedder = dashscope.Qwen3_embedding
)

const (
	MAX_REACT_ITERATIONS      int    = 10
	STAGE_CHANNEL_BUFFER_SIZE int    = 5
	PINECONE_INDEX_NAME       string = "dubbot"
	MCP_HOST_NAME             string = "mcp_host"
	K8S_RAG_INDEX             string = "kube-docs"
	RAG_TOP_K                 int    = 10
	RERANK_TOP_N              int    = 2
)
