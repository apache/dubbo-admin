package ragtest

import (
	"context"
	"strings"
	"sync"
	"testing"

	compRag "dubbo-admin-ai/component/rag"

	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/components/indexer"
	einoRetriever "github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)

type workflowStore struct {
	mu   sync.RWMutex
	docs map[string][]*schema.Document
}

func newWorkflowStore() *workflowStore {
	return &workflowStore{docs: make(map[string][]*schema.Document)}
}

type workflowIndexer struct {
	store     *workflowStore
	lastCount int
}

func (w *workflowIndexer) Store(ctx context.Context, docs []*schema.Document, opts ...indexer.Option) ([]string, error) {
	impl := indexer.GetImplSpecificOptions(&compRag.CommonIndexerOptions{}, opts...)
	ns := impl.Namespace
	w.store.mu.Lock()
	w.store.docs[ns] = append(w.store.docs[ns], docs...)
	w.store.mu.Unlock()
	w.lastCount = len(docs)
	ids := make([]string, len(docs))
	for i := range docs {
		ids[i] = docs[i].ID
	}
	return ids, nil
}

type workflowRetriever struct{ store *workflowStore }

func (w *workflowRetriever) Retrieve(ctx context.Context, query string, opts ...einoRetriever.Option) ([]*schema.Document, error) {
	impl := einoRetriever.GetImplSpecificOptions(&compRag.CommonRetrieverOptions{}, opts...)
	ns := impl.Namespace
	w.store.mu.RLock()
	defer w.store.mu.RUnlock()
	all := w.store.docs[ns]
	out := make([]*schema.Document, 0)
	for _, d := range all {
		if strings.Contains(strings.ToLower(d.Content), strings.ToLower(query)) {
			out = append(out, d)
		}
	}
	return out, nil
}

type workflowSplitter struct{}

func (w *workflowSplitter) Transform(ctx context.Context, src []*schema.Document, opts ...document.TransformerOption) ([]*schema.Document, error) {
	if len(src) == 0 {
		return src, nil
	}
	return []*schema.Document{{ID: "c1", Content: src[0].Content[:len(src[0].Content)/2]}, {ID: "c2", Content: src[0].Content[len(src[0].Content)/2:]}}, nil
}

func TestRAGWorkflow(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "index_retrieve",
			run: func(t *testing.T) {
				ctx := context.Background()
				store := newWorkflowStore()
				r := &compRag.RAG{Indexer: &workflowIndexer{store: store}, Retriever: &workflowRetriever{store: store}}
				_, err := r.Index(ctx, "ns", []*schema.Document{{ID: "1", Content: "Dubbo is RPC"}})
				if err != nil {
					t.Fatalf("Index() error: %v", err)
				}
				got, err := r.Retrieve(ctx, "ns", []string{"RPC"})
				if err != nil {
					t.Fatalf("Retrieve() error: %v", err)
				}
				if len(got["RPC"]) == 0 || !strings.Contains(got["RPC"][0].Content, "Dubbo") {
					t.Fatalf("expected retrieval to include Dubbo, got %+v", got)
				}
			},
		},
		{
			name: "split_index",
			run: func(t *testing.T) {
				ctx := context.Background()
				idx := &workflowIndexer{store: newWorkflowStore()}
				r := &compRag.RAG{Splitter: &workflowSplitter{}, Indexer: idx}
				split, err := r.Split(ctx, []*schema.Document{{ID: "doc", Content: "abcdefghijklmnopqrstuvwxyz"}})
				if err != nil {
					t.Fatalf("Split() error: %v", err)
				}
				if len(split) <= 1 {
					t.Fatalf("expected split chunks > 1, got %d", len(split))
				}
				if _, err := r.Index(ctx, "ns", split); err != nil {
					t.Fatalf("Index() error: %v", err)
				}
				if idx.lastCount != len(split) {
					t.Fatalf("indexed count = %d, want %d", idx.lastCount, len(split))
				}
			},
		},
		{
			name: "namespace",
			run: func(t *testing.T) {
				ctx := context.Background()
				store := newWorkflowStore()
				r := &compRag.RAG{Indexer: &workflowIndexer{store: store}, Retriever: &workflowRetriever{store: store}}
				_, _ = r.Index(ctx, "ns1", []*schema.Document{{ID: "1", Content: "alpha only"}})
				_, _ = r.Index(ctx, "ns2", []*schema.Document{{ID: "2", Content: "beta only"}})
				got, err := r.Retrieve(ctx, "ns1", []string{"beta"})
				if err != nil {
					t.Fatalf("Retrieve() error: %v", err)
				}
				if len(got["beta"]) != 0 {
					t.Fatalf("expected namespace isolation, got %+v", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { tt.run(t) })
	}
}

func TestRAG_Retrieve(t *testing.T) {
	r := &compRag.RAG{Retriever: &workflowRetriever{store: newWorkflowStore()}}
	got, err := r.Retrieve(context.Background(), "ns", nil)
	if err != nil {
		t.Fatalf("Retrieve() error: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("expected non-nil empty map, got %+v", got)
	}
}
