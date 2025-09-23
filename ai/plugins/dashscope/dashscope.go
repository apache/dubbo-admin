package dashscope

import (
	"context"
	"os"

	"dubbo-admin-ai/plugins/model"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai"
	openaiGo "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const provider = "dashscope"
const baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"

const (
	qwen3_235b_a22b = "qwen3-235b-a22b-instruct-2507"
	qwen_max        = "qwen-max"
	qwen_plus       = "qwen-plus"
	qwen_flash      = "qwen-flash"
	qwen3_coder     = "qwen3-coder-plus"

	text_embedding_v4 = "text-embedding-v4"
)

type TextEmbeddingConfig struct {
	Dimensions     int                                       `json:"dimensions,omitempty"`
	EncodingFormat openaiGo.EmbeddingNewParamsEncodingFormat `json:"encodingFormat,omitempty"`
}

var (
	Qwen3       = model.NewModel(provider, qwen3_235b_a22b, &compat_oai.BasicText)
	Qwen_plus   = model.NewModel(provider, qwen_plus, &compat_oai.BasicText)
	Qwen_max    = model.NewModel(provider, qwen_max, &compat_oai.BasicText)
	Qwen3_coder = model.NewModel(provider, qwen3_coder, &compat_oai.BasicText)
	Qwen_flash  = model.NewModel(provider, qwen_flash, &compat_oai.BasicText)

	Qwen3_embedding = model.NewEmbedder(
		provider,
		text_embedding_v4,
		1024,
		&ai.EmbedderSupports{
			Input: []string{"text"},
		},
		TextEmbeddingConfig{},
	)

	supportedModels = []*model.Model{
		Qwen3,
		Qwen_plus,
		Qwen_max,
		Qwen3_coder,
		Qwen_flash,
	}

	supportedEmbeddingModels = []*model.Embedder{
		Qwen3_embedding,
	}
)

type DashScope struct {
	APIKey string

	Opts []option.RequestOption

	openAICompatible *compat_oai.OpenAICompatible
}

// Name implements genkit.Plugin.
func (o *DashScope) Name() string {
	return provider
}

// Init implements genkit.Plugin.
func (o *DashScope) Init(ctx context.Context) []api.Action {
	apiKey := o.APIKey

	// if api key is not set, get it from environment variable
	if apiKey == "" {
		apiKey = os.Getenv("DASHSCOPE_API_KEY")
	}

	if apiKey == "" {
		panic("DashScope plugin initialization failed: apiKey is required")
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
		actions = append(actions, o.DefineModel(model.InternalKey(), model.Options()).(api.Action))
	}

	// define default embedders
	for _, embedder := range supportedEmbeddingModels {
		actions = append(actions, o.DefineEmbedder(embedder.InternalKey(), embedder.Options()).(api.Action))
	}
	return actions
}

func (o *DashScope) Model(g *genkit.Genkit, name string) ai.Model {
	return o.openAICompatible.Model(g, api.NewName(provider, name))
}

func (o *DashScope) DefineModel(id string, opts ai.ModelOptions) ai.Model {
	return o.openAICompatible.DefineModel(provider, id, opts)
}

func (o *DashScope) DefineEmbedder(id string, opts *ai.EmbedderOptions) ai.Embedder {
	return o.openAICompatible.DefineEmbedder(provider, id, opts)
}

func (o *DashScope) Embedder(g *genkit.Genkit, name string) ai.Embedder {
	return o.openAICompatible.Embedder(g, api.NewName(provider, name))
}

func (o *DashScope) ListActions(ctx context.Context) []api.ActionDesc {
	return o.openAICompatible.ListActions(ctx)
}

func (o *DashScope) ResolveAction(atype api.ActionType, name string) api.Action {
	return o.openAICompatible.ResolveAction(atype, name)
}
