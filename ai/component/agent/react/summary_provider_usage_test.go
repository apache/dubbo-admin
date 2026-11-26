/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package react

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	comp "dubbo-admin-ai/component/context"
	"dubbo-admin-ai/component/hooks"
	"dubbo-admin-ai/component/memory/summary"
	"dubbo-admin-ai/providerusage"
	"dubbo-admin-ai/security"
	"dubbo-admin-ai/store"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai"
	"github.com/openai/openai-go/option"
)

// The summary path calls the model directly, so it must observe the provider
// boundary itself: the pinned adapter allocates an all-zero Usage when the
// response omits usage, which must not be accounted as provider-reported zero.
func TestSummaryAuxiliaryUsageObservation(t *testing.T) {
	for _, tc := range []struct {
		name, usage          string
		known                bool
		input, output, total int
	}{
		{name: "omitted"},
		{name: "null", usage: `,"usage":null`},
		{name: "empty", usage: `,"usage":{}`},
		{name: "partial", usage: `,"usage":{"total_tokens":5}`},
		{name: "zero", usage: `,"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}`, known: true},
		{name: "present", usage: `,"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}`, known: true, input: 2, output: 3, total: 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"id":"test","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"{}"},"finish_reason":"stop"}]%s}`, tc.usage)
			}))
			defer server.Close()
			plugin := &compat_oai.OpenAICompatible{Provider: "fixture", Opts: []option.RequestOption{
				option.WithAPIKey("test"), option.WithBaseURL(server.URL + "/v1"), option.WithMiddleware(providerusage.Middleware),
			}}
			g := genkit.Init(context.Background(), genkit.WithPlugins(plugin))
			model := plugin.DefineModel("fixture", "model", ai.ModelOptions{Supports: &compat_oai.BasicText})
			genkit.RegisterAction(g, model)

			tpl := comp.Template{ID: "conversation-summary", Version: summary.TemplateVersion, Shared: summary.SystemPrompt}
			c, err := comp.New(tpl)
			if err != nil {
				t.Fatal(err)
			}
			if err := c.Register(comp.SnapshotSource{Kind: "input"}, comp.SourcePolicy{Required: true, Kinds: []string{"user"}}); err != nil {
				t.Fatal(err)
			}
			recorder := &providerUsageRecorder{}
			manager := hooks.NewManager(nil)
			if err := manager.Register(hooks.NewUsageAuditRegistration(recorder, nil)); err != nil {
				t.Fatal(err)
			}
			ra := &ReActAgent{
				registry:        g,
				summaryCompiler: c,
				compilerSpec:    AgentSpec{ContextTokens: 4096, SafetyMargin: 100, MaxContextBytes: 1 << 20, MaxCollectBytes: 1 << 20},
				hookManager:     manager,
				model:           "fixture/model",
			}
			access := security.MustAccessContext(security.Subject{TenantID: "tenant", UserID: "user", AuthMethod: security.AuthMethodJWT})
			ctx := security.WithAccessContext(context.Background(), access)
			if _, err := ra.generateSummary(ctx, summary.GenerateRequest{
				ActiveTurn: 1, Access: access, SessionID: "session", InteractionID: "interaction", Model: "fixture/model",
				Batch: store.SummaryBatch{Source: "user:turn:session:1:record:0:offset:0", Text: "goal"}, OutputTokens: 128,
			}); err != nil {
				t.Fatal(err)
			}
			if len(recorder.events) != 1 {
				t.Fatalf("usage writes = %d", len(recorder.events))
			}
			got := recorder.events[0]
			if got.CallType != "summary" || !got.Auxiliary {
				t.Fatalf("auxiliary usage event = %+v", got)
			}
			if tc.known {
				if got.AmountSource != store.UsageAmountProvider || got.InputTokens == nil || got.OutputTokens == nil || got.TotalTokens == nil ||
					*got.InputTokens != tc.input || *got.OutputTokens != tc.output || *got.TotalTokens != tc.total {
					t.Fatalf("provider usage = %+v", got)
				}
				return
			}
			if got.AmountSource != store.UsageAmountUnknown || got.InputTokens != nil || got.OutputTokens != nil || got.TotalTokens != nil {
				t.Fatalf("missing usage must remain unknown: %+v", got)
			}
		})
	}
}
