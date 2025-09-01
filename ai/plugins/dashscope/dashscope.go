package dashscope

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

const provider = "dashscope"
const baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"

const (
	qwen3_235b_a22b = "qwen3-235b-a22b-instruct-2507"
	qwen_max        = "qwen-max"
	qwen_plus       = "qwen-plus"
	qwen_flash      = "qwen-flash"
	qwen3_coder     = "qwen3-coder-plus"

	Qwen3       = provider + "/" + qwen3_235b_a22b
	Qwen_plus   = provider + "/" + qwen_plus
	Qwen_max    = provider + "/" + qwen_max
	Qwen3_coder = provider + "/" + qwen3_coder
	Qwen_flash  = provider + "/" + qwen_flash
)

var (
	supportedModels = map[string]ai.ModelInfo{
		qwen3_235b_a22b: {
			Label:    "qwen3-235b-a22b-instruct-2507",
			Supports: &compat_oai.BasicText,
			Versions: []string{"qwen3-235b-a22b-instruct-2507"},
		},
		qwen_plus: {
			Label:    "qwen-plus",
			Supports: &compat_oai.BasicText,
			Versions: []string{"qwen-plus"},
		},
		qwen_max: {
			Label:    "qwen-max",
			Supports: &compat_oai.BasicText,
			Versions: []string{"qwen-max"},
		},
		qwen3_coder: {
			Label:    "qwen3-coder",
			Supports: &compat_oai.BasicText,
			Versions: []string{"qwen3-coder"},
		},
		qwen_flash: {
			Label:    "qwen-flash",
			Supports: &compat_oai.BasicText,
			Versions: []string{"qwen-flash"},
		},
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
