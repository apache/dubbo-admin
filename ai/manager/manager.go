package manager

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"dubbo-admin-ai/config"
	"dubbo-admin-ai/plugins/dashscope"
	"dubbo-admin-ai/plugins/siliconflow"
	"dubbo-admin-ai/utils"

	"github.com/dusted-go/logging/prettylog"
	"github.com/firebase/genkit/go/core/api"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/firebase/genkit/go/plugins/pinecone"
	"github.com/joho/godotenv"
)

var (
	gloRegistry *genkit.Genkit
	gloLogger   *slog.Logger
	once        sync.Once
)

func Registry(modelName string, logger *slog.Logger) (registry *genkit.Genkit) {
	once.Do(func() {
		var err error
		gloLogger = logger
		if logger == nil {
			gloLogger = ProductionLogger()
		}
		gloRegistry = defaultRegistry(modelName)
		if err != nil {
			panic(err)
		}
	})
	if gloRegistry == nil {
		panic("Failed to get global registry")
	}
	return gloRegistry
}

// Load environment variables from PROJECT_ROOT/.env file
func LoadEnvVars2Config() {
	dotEnvFilePath := filepath.Join(config.PROJECT_ROOT, ".env")
	dotEnvExampleFilePath := filepath.Join(config.PROJECT_ROOT, ".env.example")

	// Check if the .env file exists, if not, copy .env.example to .env
	if _, err := os.Stat(dotEnvFilePath); os.IsNotExist(err) {
		if err = utils.CopyFile(dotEnvExampleFilePath, dotEnvFilePath); err != nil {
			panic(err)
		}
	}

	// Load environment variables
	if err := godotenv.Load(dotEnvFilePath); err != nil {
		panic(err)
	}

	config.GEMINI_API_KEY = os.Getenv("GEMINI_API_KEY")
	config.SILICONFLOW_API_KEY = os.Getenv("SILICONFLOW_API_KEY")
	config.DASHSCOPE_API_KEY = os.Getenv("DASHSCOPE_API_KEY")
	config.PINECONE_API_KEY = os.Getenv("PINECONE_API_KEY")
	config.COHERE_API_KEY = os.Getenv("COHERE_API_KEY")
}

func defaultRegistry(modelName string) *genkit.Genkit {
	LoadEnvVars2Config()
	ctx := context.Background()
	plugins := []api.Plugin{}
	if config.SILICONFLOW_API_KEY != "" {
		plugins = append(plugins, &siliconflow.SiliconFlow{
			APIKey: config.SILICONFLOW_API_KEY,
		})
	}
	if config.GEMINI_API_KEY != "" {
		plugins = append(plugins, &googlegenai.GoogleAI{
			APIKey: config.GEMINI_API_KEY,
		})
	}
	if config.DASHSCOPE_API_KEY != "" {
		plugins = append(plugins, &dashscope.DashScope{
			APIKey: config.DASHSCOPE_API_KEY,
		})
	}
	if config.PINECONE_API_KEY != "" {
		plugins = append(plugins, &pinecone.Pinecone{
			APIKey: config.PINECONE_API_KEY,
		})
	}

	registry := genkit.Init(ctx,
		genkit.WithPlugins(plugins...),
		genkit.WithDefaultModel(modelName),
		genkit.WithPromptDir(config.PROMPT_DIR_PATH),
	)
	return registry
}

func DevLogger() *slog.Logger {
	slog.SetDefault(
		slog.New(
			prettylog.NewHandler(&slog.HandlerOptions{
				Level:       slog.LevelDebug,
				AddSource:   true,
				ReplaceAttr: nil,
			}),
		),
	)
	return slog.Default()
}

func ProductionLogger() *slog.Logger {
	slog.SetDefault(
		slog.New(
			prettylog.NewHandler(&slog.HandlerOptions{
				Level:       slog.LevelInfo,
				AddSource:   false,
				ReplaceAttr: nil,
			}),
		),
	)
	return slog.Default()
}

func GetLogger() *slog.Logger {
	if gloLogger == nil {
		gloLogger = ProductionLogger()
	}
	return gloLogger
}
