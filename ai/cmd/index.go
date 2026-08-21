package main

import (
	"context"
	compRag "dubbo-admin-ai/component/rag"
	appconfig "dubbo-admin-ai/config"
	"dubbo-admin-ai/runtime"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/firebase/genkit/go/core/api"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/firebase/genkit/go/plugins/pinecone"
	"github.com/openai/openai-go/option"
	"gopkg.in/yaml.v3"
)

type IndexCommand struct {
	Directory  string
	ConfigPath string
	IndexName  string
}

func main() {
	cmd := parseFlags()

	if err := validateCommand(cmd); err != nil {
		log.Fatalf("Validation error: %v", err)
	}

	if err := executeIndexing(cmd); err != nil {
		log.Fatalf("Indexing failed: %v", err)
	}

	fmt.Println("Indexing completed successfully")
}

func parseFlags() *IndexCommand {
	cmd := &IndexCommand{}

	flag.StringVar(&cmd.Directory, "dir", "component/rag/seeds", "Directory to index (required)")
	flag.StringVar(&cmd.ConfigPath, "config", "component/rag/rag.yaml", "Configuration file path")
	flag.StringVar(&cmd.IndexName, "index", "default", "Target index name (must match the runtime retriever target)")

	flag.Parse()
	return cmd
}

func validateCommand(cmd *IndexCommand) error {
	if cmd.Directory == "" {
		return fmt.Errorf("directory parameter is required")
	}

	// Check if directory exists
	info, err := os.Stat(cmd.Directory)
	if err != nil {
		return fmt.Errorf("failed to access directory %s: %w", cmd.Directory, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", cmd.Directory)
	}

	// Check if config file exists
	if _, err := os.Stat(cmd.ConfigPath); err != nil {
		return fmt.Errorf("failed to access config file %s: %w", cmd.ConfigPath, err)
	}

	return nil
}

func executeIndexing(cmd *IndexCommand) error {
	ctx := context.Background()

	// Load configuration from YAML file
	cfg, err := loadRAGConfig(cmd.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid rag config: %w", err)
	}

	// Initialize Genkit registry independently
	plugins := []api.Plugin{}
	// Initialize plugins from environment variables
	if key := os.Getenv("DASHSCOPE_API_KEY"); key != "" {
		plugins = append(plugins, &compat_oai.OpenAICompatible{
			Provider: "dashscope",
			Opts: []option.RequestOption{
				option.WithAPIKey(key),
				option.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
			},
		})
	}
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		plugins = append(plugins, &googlegenai.GoogleAI{APIKey: key})
	}
	if key := os.Getenv("SILICONFLOW_API_KEY"); key != "" {
		plugins = append(plugins, &compat_oai.OpenAICompatible{
			Provider: "siliconflow",
			Opts: []option.RequestOption{
				option.WithAPIKey(key),
				option.WithBaseURL("https://api.siliconflow.cn/v1"),
			},
		})
	}
	if key := os.Getenv("PINECONE_API_KEY"); key != "" {
		plugins = append(plugins, &pinecone.Pinecone{APIKey: key})
	}

	g := genkit.Init(ctx, genkit.WithPlugins(plugins...))

	// Build-time target index selection. Default to "default" so the indexed data
	// lands in the same target the runtime retriever reads from; fall back to the
	// directory name only when explicitly cleared.
	targetIndex := getNamespace(cmd.IndexName, cmd.Directory)

	sys, err := buildRAGSystem(g, cfg)
	if err != nil {
		return fmt.Errorf("failed to build RAG system: %w", err)
	}

	// Load documents via the built loader
	docs, err := compRag.LoadDirectory(ctx, sys.Loader, cmd.Directory)
	if err != nil {
		return fmt.Errorf("failed to load directory: %w", err)
	}

	fmt.Printf("Loaded %d documents from %s\n", len(docs), cmd.Directory)

	if len(docs) == 0 {
		fmt.Printf("No supported documents found in directory: %s\n", cmd.Directory)
		return nil
	}

	// Split documents (semantic/header or recursive based on config)
	splitDocs, err := sys.Split(ctx, docs)
	if err != nil {
		return fmt.Errorf("failed to split documents: %w", err)
	}

	// Index documents (namespace is a per-call runtime parameter)
	namespace := targetIndex
	if _, err := sys.Index(ctx, namespace, splitDocs, compRag.WithIndexerTargetIndex(targetIndex)); err != nil {
		return fmt.Errorf("failed to index documents: %w", err)
	}

	return nil
}

// getNamespace generates a namespace from the provided parameter or directory name
func getNamespace(namespace, directory string) string {
	if namespace != "" {
		return namespace
	}
	return filepath.Base(strings.TrimSpace(directory))
}

// loadRAGConfig loads RAG configuration from a YAML file
func loadRAGConfig(configPath string) (*compRag.RAGSpec, error) {
	loader := appconfig.NewLoader("config.yaml")
	componentCfg, err := loader.LoadComponent(configPath)
	if err != nil {
		return nil, err
	}
	if componentCfg.Type != "rag" {
		return nil, fmt.Errorf("structural error: component type must be rag, got %s", componentCfg.Type)
	}

	var cfg compRag.RAGSpec
	if err := componentCfg.Spec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode rag config: %w", err)
	}

	return &cfg, nil
}

func buildRAGSystem(g *genkit.Genkit, cfg *compRag.RAGSpec) (*compRag.RAG, error) {
	if g == nil {
		return nil, fmt.Errorf("genkit registry is nil")
	}
	if cfg == nil {
		return nil, fmt.Errorf("rag config is nil")
	}

	var specNode yaml.Node
	if err := specNode.Encode(cfg); err != nil {
		return nil, fmt.Errorf("failed to encode rag config: %w", err)
	}

	compRaw, err := compRag.RAGFactory(&specNode)
	if err != nil {
		return nil, fmt.Errorf("failed to create rag component: %w", err)
	}

	ragComp, ok := compRaw.(*compRag.RAGComponent)
	if !ok {
		return nil, fmt.Errorf("invalid rag component type: %T", compRaw)
	}

	rt := runtime.NewRuntime()
	rt.SetGenkitRegistry(g)
	if err := ragComp.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate rag component: %w", err)
	}
	if err := ragComp.Init(rt); err != nil {
		return nil, fmt.Errorf("failed to init rag component: %w", err)
	}
	if ragComp.Rag == nil {
		return nil, fmt.Errorf("rag system is not initialized")
	}

	return ragComp.Rag, nil
}
