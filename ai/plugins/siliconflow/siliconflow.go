package siliconflow

import (
	"context"
	"dubbo-admin-ai/plugins/models"
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
)

var (
	DeepSeekV3 = models.New(provider, deepseekV3, compat_oai.BasicText)
	QwenQwQ32B = models.New(provider, qwenQwQ32B, compat_oai.BasicText)
	Qwen3Coder = models.New(provider, qwen3Coder, compat_oai.Multimodal)
	DeepSeekR1 = models.New(provider, deepseekR1, compat_oai.BasicText)

	supportedModels = []models.Model{
		DeepSeekV3,
		QwenQwQ32B,
		Qwen3Coder,
		DeepSeekR1,
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
	for _, m := range supportedModels {
		if _, err := o.DefineModel(g, m.Key(), m.Info()); err != nil {
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
