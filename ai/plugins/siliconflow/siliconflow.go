package siliconflow

import (
	"context"
	"fmt"
	"os"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai"
	"github.com/openai/openai-go/option"
)

const provider = "siliconflow"
const baseURL = "https://api.siliconflow.cn/v1"

const (
	deepseekV3 = "deepseek-ai/DeepSeek-V3"
	deepseekR1 = "deepseek-ai/DeepSeek-R1"
	qwenQwQ32B = "Qwen/QwQ-32B"
	qwen3Coder = "Qwen/Qwen3-Coder-480B-A35B-Instruct"

	DeepSeekV3 = provider + "/" + deepseekV3
	QwenQwQ32B = provider + "/" + qwenQwQ32B
	Qwen3Coder = provider + "/" + qwen3Coder
	DeepSeekR1 = provider + "/" + deepseekR1
)

var (
	supportedModels = map[string]ai.ModelInfo{
		deepseekV3: {
			Label:    "deepseek-ai/DeepSeek-V3",
			Supports: &compat_oai.BasicText,
			Versions: []string{"DeepSeek-V3-0324"},
		},
		qwen3Coder: {
			Label:    "Qwen/Qwen3-Coder-480B-A35B-Instruct",
			Supports: &compat_oai.Multimodal,
			Versions: []string{"Qwen3-Coder-480B-A35B"},
		},
		qwenQwQ32B: {
			Label:    "Qwen/QwQ-32B",
			Supports: &compat_oai.BasicText,
			Versions: []string{"QwQ-32B"},
		},
		deepseekR1: {
			Label:    "deepseek-ai/DeepSeek-R1",
			Supports: &compat_oai.BasicText,
			Versions: []string{"DeepSeek-R1-0528"},
		},
	}

	knownEmbedders = []string{}
)

type SiliconFlow struct {
	APIKey string

	Opts []option.RequestOption

	openAICompatible *compat_oai.OpenAICompatible
}

// Name implements genkit.Plugin.
func (o *SiliconFlow) Name() string {
	return provider
}

// Init implements genkit.Plugin.
func (o *SiliconFlow) Init(ctx context.Context, g *genkit.Genkit) error {
	apiKey := o.APIKey

	// if api key is not set, get it from environment variable
	if apiKey == "" {
		apiKey = os.Getenv("SILICONFLOW_API_KEY")
	}

	if apiKey == "" {
		return fmt.Errorf("siliconflow plugin initialization failed: apiKey is required")
	}

	if o.openAICompatible == nil {
		o.openAICompatible = &compat_oai.OpenAICompatible{}
	}

	// set the options
	o.openAICompatible.Opts = []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	}

	if len(o.Opts) > 0 {
		o.openAICompatible.Opts = append(o.openAICompatible.Opts, o.Opts...)
	}

	o.openAICompatible.Provider = provider
	if err := o.openAICompatible.Init(ctx, g); err != nil {
		return err
	}

	// define default models
	for model, info := range supportedModels {
		if _, err := o.DefineModel(g, model, info); err != nil {
			return err
		}
	}

	// define default embedders
	for _, embedder := range knownEmbedders {
		if _, err := o.DefineEmbedder(g, embedder); err != nil {
			return err
		}
	}

	return nil
}

func (o *SiliconFlow) Model(g *genkit.Genkit, name string) ai.Model {
	return o.openAICompatible.Model(g, name, provider)
}

func (o *SiliconFlow) DefineModel(g *genkit.Genkit, name string, info ai.ModelInfo) (ai.Model, error) {
	return o.openAICompatible.DefineModel(g, provider, name, info)
}

func (o *SiliconFlow) DefineEmbedder(g *genkit.Genkit, name string) (ai.Embedder, error) {
	return o.openAICompatible.DefineEmbedder(g, provider, name)
}

func (o *SiliconFlow) Embedder(g *genkit.Genkit, name string) ai.Embedder {
	return o.openAICompatible.Embedder(g, name, provider)
}

func (o *SiliconFlow) ListActions(ctx context.Context) []core.ActionDesc {
	return o.openAICompatible.ListActions(ctx)
}

func (o *SiliconFlow) ResolveAction(g *genkit.Genkit, atype core.ActionType, name string) error {
	return o.openAICompatible.ResolveAction(g, atype, name)
}
