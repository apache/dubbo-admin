package rag

import (
	"context"
	"dubbo-admin-ai/runtime"
	"dubbo-admin-ai/utils"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/localvec"
	"github.com/firebase/genkit/go/plugins/pinecone"
)

// retrieverComponent Retriever 组件包装器
type retrieverComponent struct {
	retrieverType string
	embedderModel string
	targetIndex   string
	defaultTopK   int
	retriever     retriever.Retriever
}

func NewRetrieverComponent(retrieverType, embedderModel string, targetIndex string, defaultTopK int) (runtime.Component, error) {
	return &retrieverComponent{
		retrieverType: retrieverType,
		embedderModel: embedderModel,
		targetIndex:   targetIndex,
		defaultTopK:   defaultTopK,
	}, nil
}

func (c *retrieverComponent) Name() string { return "retriever" }

func (c *retrieverComponent) Validate() error { return nil }

func (c *retrieverComponent) Init(rt *runtime.Runtime) error {
	registry := rt.GetGenkitRegistry()
	if registry == nil {
		return fmt.Errorf("genkit registry not initialized")
	}

	rtv, err := newRetrieverByType(registry, c.retrieverType, c.embedderModel, c.targetIndex, c.defaultTopK)
	if err != nil {
		return fmt.Errorf("failed to create retriever: %w", err)
	}
	c.retriever = rtv

	rt.GetLogger().Info("Retriever component initialized",
		"type", c.retrieverType,
		"embedder", c.embedderModel,
		"target_index", c.targetIndex,
		"default_top_k", c.defaultTopK,
	)
	return nil
}

func (c *retrieverComponent) Start() error { return nil }

func (c *retrieverComponent) Stop() error { return nil }

func (c *retrieverComponent) get() retriever.Retriever {
	return c.retriever
}

// --- Retriever ---
type PineconeRetriever struct {
	g         *genkit.Genkit
	embedder  string
	target    string
	defaultK  int
	retriever map[string]ai.Retriever // keyed by target index
}

func newPineconeRetriever(g *genkit.Genkit, embedderModel string, targetIndex string, topK int) *PineconeRetriever {
	return &PineconeRetriever{
		g:        g,
		embedder: embedderModel,
		target:   targetIndex,
		defaultK: topK,
	}
}

func (r *PineconeRetriever) getRetriever(ctx context.Context, targetIndex string) (ai.Retriever, error) {
	if targetIndex == "" {
		targetIndex = "default"
	}

	if r.retriever == nil {
		r.retriever = make(map[string]ai.Retriever)
	}
	ret := r.retriever[targetIndex]
	if ret != nil {
		return ret, nil
	}

	embedder := genkit.LookupEmbedder(r.g, r.embedder)
	if embedder == nil {
		return nil, fmt.Errorf("failed to find embedder %s", r.embedder)
	}

	var err error
	if !pinecone.IsDefinedRetriever(r.g, targetIndex) {
		_, ret, err = pinecone.DefineRetriever(ctx, r.g,
			pinecone.Config{
				IndexID:  targetIndex,
				Embedder: embedder,
			},
			&ai.RetrieverOptions{
				Label:        targetIndex,
				ConfigSchema: core.InferSchemaMap(pinecone.PineconeRetrieverOptions{}),
			})
	} else {
		ret = pinecone.Retriever(r.g, targetIndex)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to define retriever: %w", err)
	}

	if r.retriever == nil {
		r.retriever = make(map[string]ai.Retriever)
	}
	if existing := r.retriever[targetIndex]; existing != nil {
		ret = existing
	} else {
		r.retriever[targetIndex] = ret
	}

	return ret, nil
}

func (r *PineconeRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	impl := retriever.GetImplSpecificOptions(&RAGOptions{}, opts...)
	effectiveTarget := r.target
	if impl.TargetIndex != nil && *impl.TargetIndex != "" {
		effectiveTarget = *impl.TargetIndex
	}
	ret, err := r.getRetriever(ctx, effectiveTarget)
	if err != nil {
		return nil, err
	}

	// Options handling
	// Default options
	defaultK := r.defaultK
	pineconeOpts := &pinecone.PineconeRetrieverOptions{
		K: defaultK, // Default TopK
	}

	// Apply Eino common options
	commonOpts := retriever.GetCommonOptions(&retriever.Options{
		TopK: &defaultK,
	}, opts...)

	if commonOpts.TopK != nil {
		pineconeOpts.K = *commonOpts.TopK
	}

	// Apply implementation specific options (for Namespace)
	if impl.Namespace != "" {
		pineconeOpts.Namespace = impl.Namespace
	}

	// Retrieve
	resp, err := ret.Retrieve(ctx, &ai.RetrieverRequest{
		Query:   ai.DocumentFromText(query, nil),
		Options: pineconeOpts,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve: %w", err)
	}

	docs := utils.ToEinoDocuments(resp.Documents)

	return docs, nil
}

type DevRetriever struct {
	g         *genkit.Genkit
	embedder  string
	target    string
	defaultK  int
	mu        sync.Mutex
	retriever map[string]ai.Retriever // keyed by target index
}

func newDevRetriever(g *genkit.Genkit, embedderModel string, targetIndex string, topK int) *DevRetriever {
	return &DevRetriever{
		g:        g,
		embedder: embedderModel,
		target:   targetIndex,
		defaultK: topK,
	}
}

func (r *DevRetriever) getRetriever(ctx context.Context, targetIndex string) (ai.Retriever, error) {
	if targetIndex == "" {
		targetIndex = "default"
	}

	r.mu.Lock()
	if r.retriever == nil {
		r.retriever = make(map[string]ai.Retriever)
	}
	ret := r.retriever[targetIndex]
	r.mu.Unlock()
	if ret != nil {
		return ret, nil
	}

	embedder := genkit.LookupEmbedder(r.g, r.embedder)
	if embedder == nil {
		return nil, fmt.Errorf("failed to find embedder %s", r.embedder)
	}

	if err := localvec.Init(); err != nil {
		return nil, fmt.Errorf("failed to init localvec: %w", err)
	}

	localvecConfig := localvec.Config{Embedder: embedder}

	var err error
	if localvec.IsDefinedRetriever(r.g, targetIndex) {
		ret = localvec.Retriever(r.g, targetIndex)
	} else {
		_, ret, err = localvec.DefineRetriever(r.g, targetIndex, localvecConfig, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to define localvec retriever: %w", err)
	}

	r.mu.Lock()
	if r.retriever == nil {
		r.retriever = make(map[string]ai.Retriever)
	}
	if existing := r.retriever[targetIndex]; existing != nil {
		ret = existing
	} else {
		r.retriever[targetIndex] = ret
	}
	r.mu.Unlock()

	return ret, nil
}

func (r *DevRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	impl := retriever.GetImplSpecificOptions(&RAGOptions{}, opts...)
	effectiveTarget := r.target
	if impl.TargetIndex != nil && *impl.TargetIndex != "" {
		effectiveTarget = *impl.TargetIndex
	}
	ret, err := r.getRetriever(ctx, effectiveTarget)
	if err != nil {
		return nil, err
	}

	// Options handling
	defaultK := r.defaultK
	// Apply Eino common options
	commonOpts := retriever.GetCommonOptions(&retriever.Options{
		TopK: &defaultK,
	}, opts...)

	k := defaultK
	if commonOpts.TopK != nil {
		k = *commonOpts.TopK
	}

	// Retrieve
	retrieverReq := &ai.RetrieverRequest{
		Query: ai.DocumentFromText(query, nil),
		Options: &localvec.RetrieverOptions{
			K: k,
		},
	}
	resp, err := ret.Retrieve(ctx, retrieverReq)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve: %w", err)
	}

	docs := utils.ToEinoDocuments(resp.Documents)

	return docs, nil
}

func newRetrieverByType(g *genkit.Genkit, retrieverType, embedderModel string, targetIndex string, defaultTopK int) (retriever.Retriever, error) {
	if targetIndex == "" {
		targetIndex = DefaultRetrieverTargetIndex
	}
	if defaultTopK <= 0 {
		defaultTopK = DefaultRetrieverTopK
	}
	switch retrieverType {
	case "dev":
		return newDevRetriever(g, embedderModel, targetIndex, defaultTopK), nil
	case "pinecone":
		return newPineconeRetriever(g, embedderModel, targetIndex, defaultTopK), nil
	default:
		return nil, fmt.Errorf("unsupported retriever type: %s", retrieverType)
	}
}
