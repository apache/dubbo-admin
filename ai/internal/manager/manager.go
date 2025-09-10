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

	"github.com/firebase/genkit/go/core/logger"
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

		defaultReg, err := defaultRegistry(modelName)
		if err != nil {
			panic(err)
		}
		registry = defaultReg

		if logger == nil {
			gloLogger = defaultLogger()
		}
	})
}

// Load environment variables from PROJECT_ROOT/.env file
func loadEnvVars() (err error) {
	dotEnvFilePath := filepath.Join(config.PROJECT_ROOT, ".env")
	dotEnvExampleFilePath := filepath.Join(config.PROJECT_ROOT, ".env.example")

	// Check if the .env file exists, if not, copy .env.example to .env
	if _, err = os.Stat(dotEnvFilePath); os.IsNotExist(err) {
		if err = utils.CopyFile(dotEnvExampleFilePath, dotEnvFilePath); err != nil {
			return err
		}
	}

	// Load environment variables
	err = godotenv.Load(dotEnvFilePath)
	return err
}

func defaultRegistry(modelName string) (*genkit.Genkit, error) {
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

func defaultLogger() *slog.Logger {
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
	logger.SetLevel(logLevel)

	slog.SetDefault(
		slog.New(
			tint.NewHandler(os.Stderr, &tint.Options{
				Level:      slog.LevelDebug,
				AddSource:  true,
				TimeFormat: time.Kitchen,
			}),
		),
	)
	return slog.Default()
}

func GetRegister() *genkit.Genkit {
	if registry == nil {
		defaultReg, err := defaultRegistry(config.DEFAULT_MODEL.Key())
		if err != nil {
			panic(err)
		}
		registry = defaultReg
	}
	return registry
}

func GetLogger() *slog.Logger {
	if gloLogger == nil {
		gloLogger = defaultLogger()
	}
	return gloLogger
}
