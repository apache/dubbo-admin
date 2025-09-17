package siliconflow

import (
	"context"
	"dubbo-admin-ai/plugins/model"
	"os"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
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
	DeepSeekV3 = model.New(provider, deepseekV3, compat_oai.BasicText)
	QwenQwQ32B = model.New(provider, qwenQwQ32B, compat_oai.BasicText)
	Qwen3Coder = model.New(provider, qwen3Coder, compat_oai.Multimodal)
	DeepSeekR1 = model.New(provider, deepseekR1, compat_oai.BasicText)

	supportedModels = []model.Model{
		DeepSeekV3,
		QwenQwQ32B,
		Qwen3Coder,
		DeepSeekR1,
	}

	// supportedEmbeddingModels = []string{}
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

func (o *SiliconFlow) Init(ctx context.Context) []api.Action {
	apiKey := o.APIKey

	// if api key is not set, get it from environment variable
	if apiKey == "" {
		apiKey = os.Getenv("SILICONFLOW_API_KEY")
	}

	if apiKey == "" {
		panic("SiliconFlow plugin initialization failed: apiKey is required")
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
	compatActions := o.openAICompatible.Init(ctx)

	var actions []api.Action
	actions = append(actions, compatActions...)

	// define default models
	for _, model := range supportedModels {
		actions = append(actions, o.DefineModel(model.Key(), model.Info()).(api.Action))
	}
	//TODO: define default embedders

	return actions
}

func (o *SiliconFlow) Model(g *genkit.Genkit, name string) ai.Model {
	return o.openAICompatible.Model(g, api.NewName(provider, name))
}

func (o *SiliconFlow) DefineModel(id string, opts ai.ModelOptions) ai.Model {
	return o.openAICompatible.DefineModel(provider, id, opts)
}

func (o *SiliconFlow) DefineEmbedder(id string, opts *ai.EmbedderOptions) ai.Embedder {
	return o.openAICompatible.DefineEmbedder(provider, id, opts)
}

func (o *SiliconFlow) Embedder(g *genkit.Genkit, name string) ai.Embedder {
	return o.openAICompatible.Embedder(g, api.NewName(provider, name))
}

func (o *SiliconFlow) ListActions(ctx context.Context) []api.ActionDesc {
	return o.openAICompatible.ListActions(ctx)
}

func (o *SiliconFlow) ResolveAction(atype api.ActionType, name string) api.Action {
	return o.openAICompatible.ResolveAction(atype, name)
}
