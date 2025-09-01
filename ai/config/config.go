package config

import (
	"dubbo-admin-ai/plugins/siliconflow"
	"os"
	"path/filepath"
	"runtime"
)

var (
	// API keys
	GEMINI_API_KEY      string = os.Getenv("GEMINI_API_KEY")
	SILICONFLOW_API_KEY string = os.Getenv("SILICONFLOW_API_KEY")

	// Configuration
	// 自动获取项目根目录
	_, b, _, _   = runtime.Caller(0)
	PROJECT_ROOT = filepath.Join(filepath.Dir(b), "..")

	PROMPT_DIR_PATH      string = filepath.Join(PROJECT_ROOT, "prompts")
	LOG_LEVEL            string = os.Getenv("LOG_LEVEL")
	DEFAULT_MODEL        string = siliconflow.DeepSeekV3
	MAX_REACT_ITERATIONS int    = 5
)
