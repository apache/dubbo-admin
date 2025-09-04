package dashscope

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

const provider = "dashscope"
const baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"

const (
	qwen3_235b_a22b = "qwen3-235b-a22b-instruct-2507"
	qwen_max        = "qwen-max"
	qwen_plus       = "qwen-plus"
	qwen_flash      = "qwen-flash"
	qwen3_coder     = "qwen3-coder-plus"
)

var (
	Qwen3       = models.New(provider, qwen3_235b_a22b, compat_oai.BasicText)
	Qwen_plus   = models.New(provider, qwen_plus, compat_oai.BasicText)
	Qwen_max    = models.New(provider, qwen_max, compat_oai.BasicText)
	Qwen3_coder = models.New(provider, qwen3_coder, compat_oai.BasicText)
	Qwen_flash  = models.New(provider, qwen_flash, compat_oai.BasicText)

	supportedModels = []models.Model{
		Qwen3,
		Qwen_plus,
		Qwen_max,
		Qwen3_coder,
		Qwen_flash,
	}

	knownEmbedders = []string{}
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
func (o *DashScope) Init(ctx context.Context, g *genkit.Genkit) error {
	apiKey := o.APIKey

	// if api key is not set, get it from environment variable
	if apiKey == "" {
		apiKey = os.Getenv("DASHSCOPE_API_KEY")
	}

	if apiKey == "" {
		return fmt.Errorf("DashScope plugin initialization failed: apiKey is required")
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
	for _, model := range supportedModels {
		if _, err := o.DefineModel(g, model.Key(), model.Info()); err != nil {
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

func (o *DashScope) Model(g *genkit.Genkit, name string) ai.Model {
	return o.openAICompatible.Model(g, name, provider)
}

func (o *DashScope) DefineModel(g *genkit.Genkit, name string, info ai.ModelInfo) (ai.Model, error) {
	return o.openAICompatible.DefineModel(g, provider, name, info)
}

func (o *DashScope) DefineEmbedder(g *genkit.Genkit, name string) (ai.Embedder, error) {
	return o.openAICompatible.DefineEmbedder(g, provider, name)
}

func (o *DashScope) Embedder(g *genkit.Genkit, name string) ai.Embedder {
	return o.openAICompatible.Embedder(g, name, provider)
}

func (o *DashScope) ListActions(ctx context.Context) []core.ActionDesc {
	return o.openAICompatible.ListActions(ctx)
}

func (o *DashScope) ResolveAction(g *genkit.Genkit, atype core.ActionType, name string) error {
	return o.openAICompatible.ResolveAction(g, atype, name)
}
