package manager

import (
	"context"
	"dubbo-admin-ai/config"
	"dubbo-admin-ai/plugins/siliconflow"
	"dubbo-admin-ai/utils"
	"fmt"
	"log"
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
	globalGenkit *genkit.Genkit
	rootContext  *context.Context
	globalLogger *slog.Logger
)

func InitGlobalGenkit(defaultModel string) (err error) {
	ctx := context.Background()
	if rootContext == nil {
		rootContext = &ctx
	}
	g, err := genkit.Init(*rootContext,
		genkit.WithPlugins(
			&siliconflow.SiliconFlow{
				APIKey: config.SILICONFLOW_API_KEY,
			},
			&googlegenai.GoogleAI{
				APIKey: config.GEMINI_API_KEY,
			},
		),
		genkit.WithDefaultModel(defaultModel),
		genkit.WithPromptDir(config.PROMPT_DIR_PATH),
	)

	if g == nil {
		return fmt.Errorf("fail to initialize global genkit")
	}

	globalGenkit = g
	return err
}

func InitLogger() {
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
	globalLogger = slog.Default()
}

func GetGlobalGenkit() (*genkit.Genkit, error) {
	var err error
	if globalGenkit == nil {
		err = InitGlobalGenkit(config.DEFAULT_MODEL)
		if err != nil {
			log.Fatalf("Failed to initialize global genkit: %v", err)
		}
	}
	return globalGenkit, err
}

func GetLogger() *slog.Logger {
	if globalLogger == nil {
		InitLogger()
	}
	return globalLogger
}

func GetRootContext() context.Context {
	ctx := context.Background()
	if rootContext == nil {
		rootContext = &ctx
	}
	return *rootContext
}

// Load environment variables from PROJECT_ROOT/.env file
func LoadEnvVars() (err error) {
	dotEnvFilePath := filepath.Join(config.PROJECT_ROOT, ".env")
	dotEnvExampleFilePath := filepath.Join(config.PROJECT_ROOT, ".env.example")

	// Check if the .env file exists，if not, copy .env.example to .env
	if _, err = os.Stat(dotEnvFilePath); os.IsNotExist(err) {
		if err = utils.CopyFile(dotEnvExampleFilePath, dotEnvFilePath); err != nil {
			return err
		}
	}

	// Load environment variables
	err = godotenv.Load(dotEnvFilePath)
	return err
}
