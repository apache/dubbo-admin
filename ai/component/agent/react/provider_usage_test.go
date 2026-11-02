package react

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"dubbo-admin-ai/component/hooks"
	"dubbo-admin-ai/providerusage"
	"dubbo-admin-ai/security"
	"dubbo-admin-ai/store"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai"
	"github.com/openai/openai-go/option"
)

type providerUsageRecorder struct{ events []store.UsageEvent }

func (r *providerUsageRecorder) RecordUsage(_ context.Context, e store.UsageEvent) error {
	r.events = append(r.events, e)
	return nil
}
func (*providerUsageRecorder) RecordAudit(context.Context, store.AuditEvent) error { return nil }

// Exercise the pinned Genkit adapter and accounting hook together: the adapter
// allocates an all-zero Usage even when the HTTP response omitted usage.
func TestProviderHTTPUsageAccounting(t *testing.T) {
	for _, streaming := range []bool{false, true} {
		for _, tc := range []struct {
			name, usage string
			known       bool
			total       int
		}{
			{name: "omitted"},
			{name: "null", usage: `,"usage":null`},
			{name: "empty", usage: `,"usage":{}`},
			{name: "partial", usage: `,"usage":{"total_tokens":5}`},
			{name: "zero", usage: `,"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}`, known: true},
			{name: "present", usage: `,"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}`, known: true, total: 5},
		} {
			t.Run(fmt.Sprintf("stream=%v/%s", streaming, tc.name), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if streaming {
						w.Header().Set("Content-Type", "text/event-stream")
						fmt.Fprint(w, "data: {\"id\":\"test\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"ok\"},\"finish_reason\":null}]}\n\n")
						// Real providers send final usage in a chunk without choices. Genkit
						// v1.2 skips that chunk's usage, so accounting must observe the wire.
						fmt.Fprintf(w, "data: {\"id\":\"test\",\"choices\":[]%s}\n\ndata: [DONE]\n\n", tc.usage)
					} else {
						w.Header().Set("Content-Type", "application/json")
						fmt.Fprintf(w, `{"id":"test","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]%s}`, tc.usage)
					}
				}))
				defer server.Close()
				plugin := &compat_oai.OpenAICompatible{Provider: "fixture", Opts: []option.RequestOption{
					option.WithAPIKey("test"), option.WithBaseURL(server.URL + "/v1"), option.WithMiddleware(providerusage.Middleware),
				}}
				g := genkit.Init(context.Background(), genkit.WithPlugins(plugin))
				model := plugin.DefineModel("fixture", "model", ai.ModelOptions{Supports: &compat_oai.BasicText})
				genkit.RegisterAction(g, model)
				prompt := genkit.DefinePrompt(g, "usage-test", ai.WithModel(model), ai.WithPrompt("hello"))
				recorder := &providerUsageRecorder{}
				manager := hooks.NewManager(nil)
				if err := manager.Register(hooks.NewUsageAuditRegistration(recorder, nil)); err != nil {
					t.Fatal(err)
				}
				ctx := security.WithAccessContext(context.Background(), security.MustAccessContext(security.Subject{TenantID: "tenant", UserID: "user", AuthMethod: security.AuthMethodJWT}))
				var opts []ai.PromptExecuteOption
				if streaming {
					opts = append(opts, ai.WithStreaming(func(context.Context, *ai.ModelResponseChunk) error { return nil }))
				}
				_, err := executeModelCall(ctx, &interactionTrace{Session: "session", InteractionID: "interaction", Model: "fixture/model", hookManager: manager}, prompt, nil, nil, opts...)
				if err != nil {
					t.Fatal(err)
				}
				if len(recorder.events) != 1 {
					t.Fatalf("usage writes = %d", len(recorder.events))
				}
				got := recorder.events[0]
				if tc.known {
					if got.AmountSource != store.UsageAmountProvider || got.TotalTokens == nil || *got.TotalTokens != tc.total || got.InputTokens == nil || got.OutputTokens == nil {
						t.Fatalf("provider usage = %+v", got)
					}
				} else if got.AmountSource != store.UsageAmountUnknown || got.InputTokens != nil || got.OutputTokens != nil || got.TotalTokens != nil {
					t.Fatalf("missing usage must remain unknown: %+v", got)
				}
			})
		}
	}
}
