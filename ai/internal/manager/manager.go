package manager

import (
	"context"
	"dubbo-admin-ai/config"
	"sync"

	"dubbo-admin-ai/plugins/dashscope"
	"dubbo-admin-ai/plugins/siliconflow"
	"dubbo-admin-ai/utils"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dusted-go/logging/prettylog"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
)

var (
	registry  *genkit.Genkit
	gloLogger *slog.Logger
	once      sync.Once
)

func Init(modelName string, logger *slog.Logger) {
	once.Do(func() {
		loadEnvVars()
		registry = defaultRegistry(modelName)
		gloLogger = logger

		if logger == nil {
			gloLogger = DevLogger()
		}
	})
}

// Load environment variables from PROJECT_ROOT/.env file
func loadEnvVars() {
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

}

func defaultRegistry(modelName string) *genkit.Genkit {
	ctx := context.Background()
	return genkit.Init(ctx,
		genkit.WithPlugins(
			&siliconflow.SiliconFlow{
				APIKey: config.SILICONFLOW_API_KEY,
			},

			&googlegenai.GoogleAI{
				APIKey: config.GEMINI_API_KEY,
			},
			&dashscope.DashScope{
				APIKey: config.DASHSCOPE_API_KEY,
			},
		),
		genkit.WithDefaultModel(modelName),
		genkit.WithPromptDir(config.PROMPT_DIR_PATH),
	)
}

func DevLogger() *slog.Logger {
	logLevel := slog.LevelInfo
	if envLevel := config.LOG_LEVEL; envLevel != "" {
		switch strings.ToUpper(envLevel) {
		case "DEBUG":
			logLevel = slog.LevelDebug
		case "INFO":
			logLevel = slog.LevelInfo
		case "WARN", "WARNING":
			logLevel = slog.LevelWarn
		case "ERROR":
			logLevel = slog.LevelError
		}
	}

	slog.SetDefault(
		slog.New(
			tint.NewHandler(os.Stderr, &tint.Options{
				Level:      logLevel,
				AddSource:  true,
				TimeFormat: time.Kitchen,
			}),
		),
	)
	return slog.Default()
}

func ReleaseLogger() *slog.Logger {
	slog.SetDefault(
		slog.New(
			tint.NewHandler(os.Stderr, &tint.Options{
				Level:      slog.LevelInfo,
				AddSource:  true,
				TimeFormat: time.Kitchen,
			}),
		),
	)
	return slog.Default()
}

func PrettyLogger() *slog.Logger {
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

func GetRegistry() *genkit.Genkit {
	if registry == nil {
		registry = defaultRegistry(config.DEFAULT_MODEL.Key())
	}
	return registry
}

func GetLogger() *slog.Logger {
	if gloLogger == nil {
		gloLogger = DevLogger()
	}
	return gloLogger
}
