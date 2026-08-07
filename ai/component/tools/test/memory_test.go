package toolstest

import (
	"context"
	"encoding/json"
	"testing"

	compRag "dubbo-admin-ai/component/rag"
	rerankers "dubbo-admin-ai/component/rag/rerankers"
	toolEngine "dubbo-admin-ai/component/tools/engine"
	"dubbo-admin-ai/runtime"

	einoRetriever "github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/firebase/genkit/go/genkit"
)

func TestK8SRAGQueryInputUnmarshalQueries(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "accept queries array",
			input: `{"queries":["nginx","deployment"]}`,
			want:  []string{"nginx", "deployment"},
		},
		{
			name:  "ignore legacy query field",
			input: `{"query":["legacy"]}`,
			want:  nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var got toolEngine.K8SRAGQueryInput
			if err := json.Unmarshal([]byte(tt.input), &got); err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}
			if len(got.Queries) != len(tt.want) {
				t.Fatalf("got len=%d, want len=%d; got=%v want=%v", len(got.Queries), len(tt.want), got.Queries, tt.want)
			}
			for i := range got.Queries {
				if got.Queries[i] != tt.want[i] {
					t.Fatalf("got[%d]=%q, want[%d]=%q", i, got.Queries[i], i, tt.want[i])
				}
			}
		})
	}
}

func TestK8SRAGQueryInputMarshalQueries(t *testing.T) {
	input := toolEngine.K8SRAGQueryInput{Queries: []string{"k8s", "service"}}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	got := string(data)
	want := `{"queries":["k8s","service"]}`
	if got != want {
		t.Fatalf("got=%s, want=%s", got, want)
	}
}

func TestK8SRAGToolOptionsDefault(t *testing.T) {
	got := (toolEngine.K8SRAGToolOptions{}).Default()
	if got.RetrieveTopK != toolEngine.DefaultK8SDocRetrieveTopK ||
		got.Namespace != toolEngine.DefaultK8SDocNamespace ||
		got.TargetIndex != toolEngine.DefaultK8SDocTargetIndex ||
		got.RerankTopN != toolEngine.DefaultK8SDocRerankTopN {
		t.Fatalf("unexpected defaults: %+v", got)
	}
}

type captureRetriever struct {
	topK        int
	namespace   string
	targetIndex string
}

func (c *captureRetriever) Retrieve(ctx context.Context, query string, opts ...einoRetriever.Option) ([]*schema.Document, error) {
	common := einoRetriever.GetCommonOptions(&einoRetriever.Options{}, opts...)
	if common.TopK != nil {
		c.topK = *common.TopK
	}
	impl := einoRetriever.GetImplSpecificOptions(&compRag.CommonRetrieverOptions{}, opts...)
	c.namespace = impl.Namespace
	if impl.TargetIndex != nil {
		c.targetIndex = *impl.TargetIndex
	}
	return []*schema.Document{{Content: "doc for " + query}}, nil
}

type captureReranker struct {
	topN int
}

func (c *captureReranker) Rerank(ctx context.Context, query string, docs any, opts ...compRag.Option) ([]*rerankers.Result, error) {
	var co rerankers.CallOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&co)
		}
	}
	if co.TopN != nil {
		c.topN = *co.TopN
	}
	return []*rerankers.Result{{Content: "reranked", RelevanceScore: 1}}, nil
}

func runRetrieveTool(t *testing.T, input map[string]any, ragSys *compRag.RAG) {
	t.Helper()
	rt := runtime.NewRuntime()
	g := genkit.Init(context.Background())
	rt.SetGenkitRegistry(g)
	rt.RegisterComponent(&compRag.RAGComponent{Rag: ragSys})
	tool := toolEngine.RetrieveBasicConceptFromK8SDoc(rt)
	raw, err := tool.RunRaw(context.Background(), input)
	if err != nil {
		t.Fatalf("RunRaw() error: %v", err)
	}
	if raw == nil {
		t.Fatalf("expected non-nil tool output")
	}
}

func TestRetrieveBasicConceptFromK8SDoc_UsesDefaultOptions(t *testing.T) {
	r := &captureRetriever{}
	rr := &captureReranker{}
	ragSys := &compRag.RAG{Retriever: r, Reranker: rr}

	runRetrieveTool(t, map[string]any{"queries": []string{"deployment"}}, ragSys)
	// RAG.Retrieve threads the namespace default to the retriever and the rerank TopN
	// default to the reranker. Retriever-side TopK/TargetIndex are not passed through
	// this path, so they are not asserted here.
	if r.namespace != toolEngine.DefaultK8SDocNamespace ||
		rr.topN != toolEngine.DefaultK8SDocRerankTopN {
		t.Fatalf("unexpected defaults applied: namespace=%q topN=%d", r.namespace, rr.topN)
	}
}
