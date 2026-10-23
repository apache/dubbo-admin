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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	comp "dubbo-admin-ai/component/context"
	"dubbo-admin-ai/component/hooks"
	"dubbo-admin-ai/schema"
	"dubbo-admin-ai/security"
	"dubbo-admin-ai/store"
	gormstore "dubbo-admin-ai/store/gorm"
	memorystore "dubbo-admin-ai/store/memory"
	"dubbo-admin-ai/testutils"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type providerFixture struct {
	summaries bool
	mu        sync.Mutex
	requests  []map[string]any
	first     string
}

func (f *providerFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var request map[string]any
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "bad request", 400)
		return
	}
	f.mu.Lock()
	f.requests = append(f.requests, request)
	index := len(f.requests)
	f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	message := `{"role":"assistant","content":"final answer"}`
	reason := "stop"
	if index == 1 && f.first != "" {
		message = f.first
		reason = "tool_calls"
	}
	if f.summaries {
		messages, _ := request["messages"].([]any)
		if len(messages) > 0 {
			firstMessage, _ := messages[0].(map[string]any)
			system, _ := firstMessage["content"].(string)
			if strings.HasPrefix(system, "Create a bounded rolling conversation summary.") {
				last, _ := messages[len(messages)-1].(map[string]any)
				content, _ := last["content"].(string)
				if parts, ok := last["content"].([]any); ok {
					for _, part := range parts {
						p, _ := part.(map[string]any)
						text, _ := p["text"].(string)
						content += text
					}
				}
				var input struct {
					Previous store.SummaryPayload `json:"previous"`
					Source   string               `json:"current_source"`
					Evidence string               `json:"evidence"`
				}
				if err := json.Unmarshal([]byte(content), &input); err != nil {
					http.Error(w, "bad summary input", 400)
					return
				}
				payload := input.Previous
				if strings.Contains(input.Evidence, "initial goal: identify outage") && len(payload.Goals) == 0 {
					payload.Goals = []store.SummaryStatement{{Text: "earliest objective preserved", Sources: []string{input.Source}}}
				}
				body, _ := json.Marshal(payload)
				encoded, _ := json.Marshal(map[string]any{"role": "assistant", "content": string(body)})
				message = string(encoded)
				reason = "stop"
			}
		}
	}
	_, _ = fmt.Fprintf(w, `{"id":"response-%d","object":"chat.completion","created":1,"model":"fixture","choices":[{"index":0,"message":%s,"finish_reason":"%s"}],"usage":{"prompt_tokens":17,"completion_tokens":5,"total_tokens":22}}`, index, message, reason)
}

func compilerAgentFixture(t *testing.T, provider *providerFixture) (*ReActAgent, *memorystore.MemoryStore, *AgentSpec, *genkit.Genkit, *atomic.Int32) {
	t.Helper()
	server := httptest.NewServer(provider)
	t.Cleanup(server.Close)
	plugin := &compat_oai.OpenAICompatible{Provider: "fixture", APIKey: "test", BaseURL: server.URL + "/v1"}
	g := genkit.Init(context.Background(), genkit.WithPlugins(plugin))
	genkit.RegisterAction(g, plugin.DefineModel("fixture", "fixture", ai.ModelOptions{Supports: &compat_oai.BasicText}))
	executions := &atomic.Int32{}
	tool := genkit.DefineTool(g, "lookup", "Lookup a named resource", func(ctx *ai.ToolContext, input struct {
		Name string `json:"name"`
	}) (map[string]any, error) {
		executions.Add(1)
		if input.Name == "fail" {
			return nil, errors.New("fixture failure")
		}
		return map[string]any{"name": input.Name, "password": "never-send-this"}, nil
	})
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "policy.txt"), []byte("Shared policy {{.Language}}"), 0600); err != nil {
		t.Fatal(err)
	}
	spec := &AgentSpec{AgentType: "react", Model: "fixture/fixture", PromptBasePath: dir, PromptFile: "policy.txt", TemplateVariables: map[string]string{"Language": "Chinese"}, MaxIterations: 2, ChannelBufferSize: 16, MaxTokens: 512, Timeout: 5}
	spec.ApplyDefaults()
	s := memorystore.NewMemoryStore(10)
	createAgentSession(t, s, "compiled")
	ra, err := NewReActAgentWithStore(g, s, spec, newToolTimeoutResolver(5, nil), nil, []ai.ToolRef{tool})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ra.Stop)
	return ra, s, spec, g, executions
}

func runCompiledInteraction(t *testing.T, ra *ReActAgent, s store.BackendStore, key string) uint64 {
	t.Helper()
	ctx := testutils.ContextWithTestAccess(context.Background())
	p, err := s.PrepareInteraction(ctx, testutils.TestAccess(), "compiled", key, key, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	ctx = store.WithPreparedInteraction(ctx, p)
	channels := ra.Interact(ctx, &schema.UserInput{Content: "check the service"}, "compiled")
	for {
		select {
		case err := <-channels.ErrorChan:
			if err != nil {
				t.Fatal(err)
			}
		case <-channels.UserRespChan:
		case <-channels.Done():
			select {
			case err := <-channels.ErrorChan:
				if err != nil {
					t.Fatal(err)
				}
			default:
			}
			return p.TurnID
		case <-time.After(5 * time.Second):
			t.Fatal("compiled interaction stalled")
		}
	}
}

func TestPartialSuccessCancellationRecoveredInProviderRequest(t *testing.T) {
	for _, backend := range []string{"memory", "gorm"} {
		for _, retry := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/retry=%t", backend, retry), func(t *testing.T) {
				f := &providerFixture{first: `{"role":"assistant","tool_calls":[{"id":"call-one","type":"function","function":{"name":"lookup","arguments":"{\"name\":\"saved-result\"}"}},{"id":"call-two","type":"function","function":{"name":"block","arguments":"{}"}}]}`}
				unused, mem, spec, g, executions := compilerAgentFixture(t, f)
				unused.Stop()
				var s store.BackendStore = mem
				var reopen func() *gormstore.GormStore
				if backend == "gorm" {
					path := filepath.Join(t.TempDir(), "recovery.db")
					reopen = func() *gormstore.GormStore {
						db, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
						if err != nil {
							t.Fatal(err)
						}
						v, err := gormstore.NewGormStore(db, 10)
						if err != nil {
							t.Fatal(err)
						}
						if err = v.Migrate(context.Background()); err != nil {
							t.Fatal(err)
						}
						return v
					}
					s = reopen()
					createAgentSession(t, s, "compiled")
				}
				entered := make(chan struct{})
				var blocks atomic.Int32
				block := genkit.DefineTool(g, "block", "Blocking fixture", func(ctx *ai.ToolContext, input struct{}) (string, error) {
					blocks.Add(1)
					close(entered)
					<-ctx.Done()
					return "", ctx.Err()
				})
				refs := []ai.ToolRef{genkit.LookupTool(g, "lookup"), block}
				ra, err := NewReActAgentWithStore(g, s, spec, newToolTimeoutResolver(5, nil), nil, refs)
				if err != nil {
					t.Fatal(err)
				}
				defer ra.Stop()
				parent, cancel := context.WithCancel(testutils.ContextWithTestAccess(context.Background()))
				defer cancel()
				p, err := s.PrepareInteraction(parent, testutils.TestAccess(), "compiled", "cancel", "cancel", time.Minute)
				if err != nil {
					t.Fatal(err)
				}
				channels := ra.Interact(store.WithPreparedInteraction(parent, p), &schema.UserInput{Content: "check the service"}, "compiled")
				select {
				case <-entered:
					cancel()
				case <-time.After(5 * time.Second):
					t.Fatal("second tool did not start")
				}
				select {
				case <-channels.Done():
				case <-time.After(5 * time.Second):
					t.Fatal("cancellation did not finish")
				}
				ra.Stop()
				if executions.Load() != 1 || blocks.Load() != 1 {
					t.Fatal("expected first success and second interrupted")
				}
				if backend == "gorm" {
					if err := s.(*gormstore.GormStore).Close(); err != nil {
						t.Fatal(err)
					}
					s = reopen()
					defer s.(*gormstore.GormStore).Close()
				}
				restored, err := NewReActAgentWithStore(g, s, spec, newToolTimeoutResolver(5, nil), nil, refs)
				if err != nil {
					t.Fatal(err)
				}
				defer restored.Stop()
				key := "new-request"
				if retry {
					key = "cancel"
				}
				runCompiledInteraction(t, restored, s, key)
				f.mu.Lock()
				defer f.mu.Unlock()
				if len(f.requests) != 2 {
					t.Fatalf("requests: %d", len(f.requests))
				}
				messages := string(comp.JSON(f.requests[1]["messages"]))
				for _, want := range []string{"Recovered tool batch", "call-one", "call-two", "saved-result", "succeeded", "interrupted", "unknown"} {
					if !strings.Contains(messages, want) {
						t.Fatalf("restored Provider messages missing %q: %s", want, messages)
					}
				}
				if strings.Contains(messages, "never-send-this") || strings.Contains(messages, "tool_calls") {
					t.Fatalf("recovery leaked secrets or fabricated native calls: %s", messages)
				}
				if executions.Load() != 1 || blocks.Load() != 1 {
					t.Fatal("recovery replayed tools")
				}
			})
		}
	}
}

func TestPendingRecoveryReachesProviderAsUnknownEvidence(t *testing.T) {
	f := &providerFixture{}
	unused, s, spec, g, executions := compilerAgentFixture(t, f)
	unused.Stop()
	ctx := context.Background()
	a := testutils.TestAccess()
	p, err := s.PrepareInteraction(ctx, a, "compiled", "crashed", "crashed", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	b := store.ToolBatch{ID: "crashed-batch", TenantID: a.Subject.TenantID, OwnerID: a.Subject.UserID, SessionID: "compiled", TurnID: p.TurnID, InteractionID: p.InteractionID, Sequence: 1, Version: 1, SchemaVersion: "v1", Calls: []store.ToolCall{{ID: "uncertain-call", Name: "lookup", Input: comp.JSON(map[string]any{"name": "uncertain-resource"}), Status: "pending"}}}
	if err := s.CreateToolBatch(ctx, a, b); err != nil {
		t.Fatal(err)
	}
	// Releasing a crashed attempt cannot establish whether the pending call ran.
	if err := s.FailInteraction(ctx, a, "compiled", p.InteractionID, p.TurnID); err != nil {
		t.Fatal(err)
	}
	ra, err := NewReActAgentWithStore(g, s, spec, newToolTimeoutResolver(5, nil), nil, []ai.ToolRef{genkit.LookupTool(g, "lookup")})
	if err != nil {
		t.Fatal(err)
	}
	defer ra.Stop()
	runCompiledInteraction(t, ra, s, "after-crash")
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.requests) != 1 {
		t.Fatalf("unexpected request count: %d", len(f.requests))
	}
	messages := string(comp.JSON(f.requests[0]["messages"]))
	for _, want := range []string{"uncertain-call", "uncertain-resource", "pending", "unknown", "Do not automatically replay"} {
		if !strings.Contains(messages, want) {
			t.Fatalf("Provider missing %s: %s", want, messages)
		}
	}
	if strings.Contains(messages, "tool_calls") || executions.Load() != 0 {
		t.Fatal("pending record replayed or fabricated native pairing")
	}
}

func TestLargeToolResultFitsProviderBudgetLiveAndAfterRecovery(t *testing.T) {
	for _, backend := range []string{"memory", "gorm"} {
		t.Run(backend, func(t *testing.T) {
			f := &providerFixture{first: `{"role":"assistant","tool_calls":[{"id":"large-call","type":"function","function":{"name":"large","arguments":"{}"}}]}`}
			unused, mem, spec, g, _ := compilerAgentFixture(t, f)
			unused.Stop()
			spec.MaxContextBytes = 16 << 10
			spec.MaxCollectBytes = 256 << 10
			var s store.BackendStore = mem
			path := filepath.Join(t.TempDir(), "large.db")
			openStore := func() *gormstore.GormStore {
				db, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
				if err != nil {
					t.Fatal(err)
				}
				v, err := gormstore.NewGormStore(db, 10)
				if err != nil {
					t.Fatal(err)
				}
				if err = v.Migrate(context.Background()); err != nil {
					t.Fatal(err)
				}
				return v
			}
			if backend == "gorm" {
				s = openStore()
				createAgentSession(t, s, "compiled")
				t.Cleanup(func() { _ = s.(*gormstore.GormStore).Close() })
			}
			var executions atomic.Int32
			output := map[string]any{"log": strings.Repeat("ordinary tool log ", 6000), "password": "large-result-secret"}
			canonicalOutput := comp.JSON(map[string]any{"tool_name": "large", "summary": "Tool executed", "result": output})
			tool := genkit.DefineTool(g, "large", "Large log fixture", func(ctx *ai.ToolContext, input struct{}) (map[string]any, error) {
				executions.Add(1)
				return output, nil
			})
			makeAgent := func() *ReActAgent {
				ra, err := NewReActAgentWithStore(g, s, spec, newToolTimeoutResolver(5, nil), nil, []ai.ToolRef{tool})
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(ra.Stop)
				return ra
			}
			reopenStore := func() {
				if backend == "gorm" {
					if err := s.(*gormstore.GormStore).Close(); err != nil {
						t.Fatal(err)
					}
					s = openStore()
				}
			}
			ra := makeAgent()
			turn := runCompiledInteraction(t, ra, s, "large-live")
			c := s.(store.CompilationStore)
			rows, err := c.ToolBatches(context.Background(), testutils.TestAccess(), "compiled", turn, 16)
			if err != nil || len(rows) != 1 {
				t.Fatalf("canonical output changed: %v", err)
			}
			var storedOutput map[string]any
			if err := json.Unmarshal(rows[0].Calls[0].Output, &storedOutput); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(storedOutput, map[string]any{"tool_name": "large", "summary": "Tool executed", "result": output}) {
				t.Fatal("canonical output was truncated or changed")
			}
			ra.Stop()
			reopenStore()
			ra = makeAgent()
			runCompiledInteraction(t, ra, s, "large-history")
			// Persist a partially completed, released attempt for recovery independently
			// of completed history. Its successful output is also larger than 16 KiB.
			ctx := context.Background()
			a := testutils.TestAccess()
			p, err := s.PrepareInteraction(ctx, a, "compiled", "released", "released", time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			b := store.ToolBatch{ID: "large-released", TenantID: a.Subject.TenantID, OwnerID: a.Subject.UserID, SessionID: "compiled", TurnID: p.TurnID, InteractionID: p.InteractionID, Sequence: 1, Version: 1, SchemaVersion: "v1", Calls: []store.ToolCall{
				{ID: "recovered-done", Name: "large", Input: comp.JSON(map[string]any{}), Status: "pending"},
				{ID: "recovered-pending", Name: "large", Input: comp.JSON(map[string]any{}), Status: "pending"},
			}}
			c = s.(store.CompilationStore)
			if err := c.CreateToolBatch(ctx, a, b); err != nil {
				t.Fatal(err)
			}
			call := b.Calls[0]
			call.Status = "succeeded"
			call.Output = canonicalOutput
			if err := c.CompleteToolCall(ctx, a, "compiled", b.ID, 1, 0, call); err != nil {
				t.Fatal(err)
			}
			if err := s.FailInteraction(ctx, a, "compiled", p.InteractionID, p.TurnID); err != nil {
				t.Fatal(err)
			}
			ra.Stop()
			reopenStore()
			ra = makeAgent()
			runCompiledInteraction(t, ra, s, "large-recovery")
			rows, err = s.(store.CompilationStore).ToolBatches(ctx, a, "compiled", p.TurnID, 16)
			if err != nil || len(rows) != 1 || !reflect.DeepEqual(rows[0].Calls[0].Output, canonicalOutput) || rows[0].Calls[1].Status != "pending" {
				t.Fatalf("canonical recovery changed: %+v %v", rows, err)
			}
			f.mu.Lock()
			defer f.mu.Unlock()
			if len(f.requests) != 4 || executions.Load() != 1 {
				t.Fatalf("requests=%d, executions=%d", len(f.requests), executions.Load())
			}
			for _, index := range []int{1, 2, 3} {
				request := f.requests[index]
				messages := string(comp.JSON(request["messages"]))
				if len(comp.JSON(request["messages"]))+len(comp.JSON(request["tools"])) > spec.MaxContextBytes {
					t.Fatal("Provider content exceeded final byte budget")
				}
				if !strings.Contains(messages, "large-call") || !strings.Contains(messages, "tool_call_id") || !strings.Contains(messages, "truncated") || strings.Contains(messages, "large-result-secret") {
					t.Fatalf("invalid Provider tool view: %s", messages)
				}
			}
			last := string(comp.JSON(f.requests[3]["messages"]))
			for _, want := range []string{"Recovered tool batch", "recovered-done", "recovered-pending", "pending", "unknown"} {
				if !strings.Contains(last, want) {
					t.Fatalf("Provider recovery missing %s", want)
				}
			}
		})
	}
}

func TestCompiledReActProviderReceivesNativeBatchAndRestoredHistory(t *testing.T) {
	f := &providerFixture{first: `{"role":"assistant","tool_calls":[{"id":"call-one","type":"function","function":{"name":"lookup","arguments":"{\"name\":\"ok\"}"}},{"id":"call-two","type":"function","function":{"name":"lookup","arguments":"{\"name\":\"fail\"}"}}]}`}
	ra, s, spec, g, executions := compilerAgentFixture(t, f)
	var reports []hooks.State
	manager := hooks.NewManager(nil)
	if err := manager.Register(hooks.Registration{Events: []hooks.Event{hooks.EventModelCallEnd}, Hook: func(ctx context.Context, state hooks.State) context.Context {
		reports = append(reports, state)
		return ctx
	}}); err != nil {
		t.Fatal(err)
	}
	ra.hookManager = manager
	turn := runCompiledInteraction(t, ra, s, "first")
	if executions.Load() != 2 {
		t.Fatalf("tool executions = %d", executions.Load())
	}
	batches, err := s.ToolBatches(context.Background(), testutils.TestAccess(), "compiled", turn, 10)
	if err != nil || len(batches) != 1 || batches[0].Calls[0].Status != "succeeded" || batches[0].Calls[1].Status != "failed" {
		t.Fatalf("canonical batch: %+v %v", batches, err)
	}
	if len(reports) != 2 || reports[1].EstimatedInputTokens == 0 || reports[1].InputTokens != 17 || reports[1].EstimatorVersion == "" {
		t.Fatalf("missing independent usage estimate: %+v", reports)
	}
	f.mu.Lock()
	requests := append([]map[string]any(nil), f.requests...)
	f.mu.Unlock()
	if len(requests) != 2 {
		t.Fatalf("requests = %d", len(requests))
	}
	first := requests[0]
	second := requests[1]
	if first["max_tokens"] != float64(512) || second["max_tokens"] != float64(512) {
		t.Fatal("output reservation not forwarded")
	}
	if len(first["tools"].([]any)) != 1 {
		t.Fatal("Act tool schema missing")
	}
	toolSchema := first["tools"].([]any)[0].(map[string]any)["function"].(map[string]any)
	if toolSchema["name"] != "lookup" || toolSchema["description"] != "Lookup a named resource" || toolSchema["parameters"] == nil {
		t.Fatalf("unexpected tool schema: %+v", toolSchema)
	}
	if tools, ok := second["tools"].([]any); ok && len(tools) != 0 {
		t.Fatal("Answer retained tools")
	}
	checkProviderMessages(t, first, false, "Act Phase")
	checkProviderMessages(t, second, true, "Final-Answer Phase")
	if strings.Contains(string(comp.JSON(second)), "never-send-this") {
		t.Fatal("secret reached provider")
	}
	// Construct a fresh Agent to prove recovery comes from the Store ledger.
	tool := genkit.LookupTool(g, "lookup")
	restored, err := NewReActAgentWithStore(g, s, spec, newToolTimeoutResolver(5, nil), nil, []ai.ToolRef{tool})
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Stop()
	runCompiledInteraction(t, restored, s, "second")
	f.mu.Lock()
	third := f.requests[2]
	f.mu.Unlock()
	checkProviderMessages(t, third, true, "Act Phase")
	if executions.Load() != 2 {
		t.Fatal("restoring history replayed a tool")
	}
}

func checkProviderMessages(t *testing.T, request map[string]any, hasBatch bool, policy string) {
	t.Helper()
	messages := request["messages"].([]any)
	systems := 0
	var calls []any
	var responses []map[string]any
	for _, value := range messages {
		m := value.(map[string]any)
		if m["role"] == "system" {
			systems++
			if !strings.Contains(fmt.Sprint(m["content"]), "Shared policy Chinese") || !strings.Contains(fmt.Sprint(m["content"]), policy) {
				t.Fatalf("incorrect system: %+v", m)
			}
		}
		if v, ok := m["tool_calls"].([]any); ok {
			calls = v
		}
		if m["role"] == "tool" {
			responses = append(responses, m)
		}
	}
	if systems != 1 {
		t.Fatalf("system count: %d", systems)
	}
	if hasBatch {
		if len(calls) != 2 || len(responses) != 2 || responses[0]["tool_call_id"] != "call-one" || responses[1]["tool_call_id"] != "call-two" {
			t.Fatalf("broken provider tool pairing: %+v", messages)
		}
	} else if len(calls) != 0 || len(responses) != 0 {
		t.Fatal("unexpected tools")
	}
}

func TestMandatoryBudgetFailureMakesNoProviderCall(t *testing.T) {
	f := &providerFixture{}
	ra, s, _, _, executions := compilerAgentFixture(t, f)
	ra.compilerSpec.ContextTokens = ra.compilerSpec.MaxTokens + ra.compilerSpec.SafetyMargin + 1
	ctx := testutils.ContextWithTestAccess(context.Background())
	channels := ra.Interact(ctx, &schema.UserInput{Content: "hello"}, "compiled")
	select {
	case err := <-channels.ErrorChan:
		if !errors.Is(err, comp.ErrMandatoryBudget) {
			t.Fatalf("error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("interaction stalled")
	}
	<-channels.Done()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.requests) != 0 || executions.Load() != 0 {
		t.Fatal("mandatory overflow invoked provider or tools")
	}
	if _, err := s.BeginTurn(ctx, testutils.TestAccess(), "compiled"); err != nil {
		t.Fatalf("failed context leaked active turn: %v", err)
	}
}

func TestModelCannotExecuteUnadvertisedTool(t *testing.T) {
	f := &providerFixture{first: `{"role":"assistant","tool_calls":[{"id":"denied","type":"function","function":{"name":"lookup","arguments":"{\"name\":\"ok\"}"}}]}`}
	ra, _, _, _, executions := compilerAgentFixture(t, f)
	ra.compilerSpec.ToolRequiredScopes = map[string]string{"lookup": "tool:lookup"}
	ctx := testutils.ContextWithTestAccess(context.Background())
	channels := ra.Interact(ctx, &schema.UserInput{Content: "hello"}, "compiled")
	select {
	case err := <-channels.ErrorChan:
		if err == nil || !strings.Contains(err.Error(), "TOOL_NOT_AUTHORIZED") {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("interaction stalled")
	}
	<-channels.Done()
	if executions.Load() != 0 {
		t.Fatal("model bypassed allowed tool set")
	}
}

func TestMinimalRollbackStillCompilesToolFreeRequest(t *testing.T) {
	f := &providerFixture{}
	ra, s, _, _, executions := compilerAgentFixture(t, f)
	ra.compilerSpec.ContextCompilerMode = "minimal"
	runCompiledInteraction(t, ra, s, "minimal")
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.requests) != 1 || executions.Load() != 0 {
		t.Fatal("minimal rollback did not finish ordinary chat")
	}
	checkProviderMessages(t, f.requests[0], false, "Final-Answer Phase")
}

type failingLedger struct{ store.CompilationStore }

func (failingLedger) CreateToolBatch(context.Context, security.AccessContext, store.ToolBatch) error {
	return store.ErrResourceExhausted
}

func TestToolLedgerMustCommitBeforeExecution(t *testing.T) {
	f := &providerFixture{first: `{"role":"assistant","tool_calls":[{"id":"call","type":"function","function":{"name":"lookup","arguments":"{\"name\":\"ok\"}"}}]}`}
	ra, _, _, _, executions := compilerAgentFixture(t, f)
	ra.compilationStore = failingLedger{ra.compilationStore}
	ctx := testutils.ContextWithTestAccess(context.Background())
	channels := ra.Interact(ctx, &schema.UserInput{Content: "hello"}, "compiled")
	select {
	case err := <-channels.ErrorChan:
		if !errors.Is(err, store.ErrResourceExhausted) {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("interaction stalled")
	}
	<-channels.Done()
	if executions.Load() != 0 {
		t.Fatal("tool executed before pending batch committed")
	}
}

func TestCancellationPreservesPartialBatchWithoutReplaying(t *testing.T) {
	f := &providerFixture{first: `{"role":"assistant","tool_calls":[{"id":"call-one","type":"function","function":{"name":"block","arguments":"{}"}},{"id":"call-two","type":"function","function":{"name":"block","arguments":"{}"}}]}`}
	ra, s, spec, g, _ := compilerAgentFixture(t, f)
	entered := make(chan struct{})
	var executions atomic.Int32
	tool := genkit.DefineTool(g, "block", "Blocking fixture", func(ctx *ai.ToolContext, input struct{}) (string, error) {
		executions.Add(1)
		close(entered)
		<-ctx.Done()
		return "", ctx.Err()
	})
	var err error
	ra, err = NewReActAgentWithStore(g, s, spec, newToolTimeoutResolver(5, nil), nil, []ai.ToolRef{tool})
	if err != nil {
		t.Fatal(err)
	}
	defer ra.Stop()
	parent, cancel := context.WithCancel(testutils.ContextWithTestAccess(context.Background()))
	defer cancel()
	p, err := s.PrepareInteraction(parent, testutils.TestAccess(), "compiled", "cancel", "cancel", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	channels := ra.Interact(store.WithPreparedInteraction(parent, p), &schema.UserInput{Content: "hello"}, "compiled")
	select {
	case <-entered:
		cancel()
	case <-time.After(5 * time.Second):
		t.Fatal("tool did not start")
	}
	select {
	case <-channels.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("cancel did not release interaction")
	}
	batches, err := s.ToolBatches(context.Background(), testutils.TestAccess(), "compiled", p.TurnID, 10)
	if err != nil || len(batches) != 1 || batches[0].Calls[0].Status != "interrupted" || batches[0].Calls[1].Status != "interrupted" || executions.Load() != 1 {
		t.Fatalf("partial batch lost: %+v; executions=%d; err=%v", batches, executions.Load(), err)
	}
}

func TestPinnedTemplateRequiresExactArchivedVersion(t *testing.T) {
	f := &providerFixture{}
	ra, s, spec, g, _ := compilerAgentFixture(t, f)
	runCompiledInteraction(t, ra, s, "original")
	old := comp.Template{ID: spec.TemplateID, Version: spec.TemplateVersion, Shared: "Shared policy {{.Language}}", Act: actPolicy, Answer: finalAnswerPolicy, AllowedTools: []string{"lookup"}, Variables: map[string]comp.Variable{"Language": {Required: true, MaxBytes: 4096}}}
	session, err := s.Get(context.Background(), testutils.TestAccess(), "compiled")
	if err != nil || session.TemplateRef != old.Ref() {
		t.Fatal("unexpected original template pin")
	}
	if err := os.WriteFile(filepath.Join(spec.PromptBasePath, spec.PromptFile), []byte("Replacement policy {{.Language}} {{.Style}}"), 0600); err != nil {
		t.Fatal(err)
	}
	spec.TemplateVersion = "2"
	spec.TemplateVariables["Style"] = "concise-style"
	tool := genkit.LookupTool(g, "lookup")
	missing, err := NewReActAgentWithStore(g, s, spec, newToolTimeoutResolver(5, nil), nil, []ai.ToolRef{tool})
	if err != nil {
		t.Fatal(err)
	}
	defer missing.Stop()
	ctx := testutils.ContextWithTestAccess(context.Background())
	channels := missing.Interact(ctx, &schema.UserInput{Content: "continue"}, "compiled")
	select {
	case err := <-channels.ErrorChan:
		if !errors.Is(err, comp.ErrTemplateVersion) {
			t.Fatalf("expected explicit template failure: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("interaction stalled")
	}
	<-channels.Done()
	archive := filepath.Join(t.TempDir(), "react-v1.json")
	if err := os.WriteFile(archive, comp.JSON(old), 0600); err != nil {
		t.Fatal(err)
	}
	spec.TemplateArchiveFiles = []string{archive}
	restored, err := NewReActAgentWithStore(g, s, spec, newToolTimeoutResolver(5, nil), nil, []ai.ToolRef{tool})
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Stop()
	runCompiledInteraction(t, restored, s, "archived")
	restored.compilerSpec.ContextCompilerMode = "minimal"
	runCompiledInteraction(t, restored, s, "archived-answer")
	freshStore := memorystore.NewMemoryStore(10)
	createAgentSession(t, freshStore, "compiled")
	fresh, err := NewReActAgentWithStore(g, freshStore, spec, newToolTimeoutResolver(5, nil), nil, []ai.ToolRef{tool})
	if err != nil {
		t.Fatal(err)
	}
	defer fresh.Stop()
	runCompiledInteraction(t, fresh, freshStore, "new-version")
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.requests) != 4 {
		t.Fatalf("missing template reached provider: requests=%d", len(f.requests))
	}
	checkProviderMessages(t, f.requests[1], false, "Act Phase")
	if strings.Contains(string(comp.JSON(f.requests[1])), "Replacement policy") {
		t.Fatal("existing session silently switched template versions")
	}
	if strings.Contains(string(comp.JSON(f.requests[2])), "Replacement policy") || strings.Contains(string(comp.JSON(f.requests[2])), "concise-style") {
		t.Fatal("Answer used new template variables")
	}
	if !strings.Contains(string(comp.JSON(f.requests[3])), "Replacement policy Chinese concise-style") {
		t.Fatal("new session did not use v2 variables")
	}
}
