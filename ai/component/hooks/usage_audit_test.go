package hooks

import (
	"context"
	"testing"
	"time"

	"dubbo-admin-ai/runtime"
	"dubbo-admin-ai/security"
	conversationstore "dubbo-admin-ai/store"
)

type usageAuditRecorder struct {
	usage []conversationstore.UsageEvent
	audit []conversationstore.AuditEvent
}

func (r *usageAuditRecorder) RecordUsage(_ context.Context, event conversationstore.UsageEvent) error {
	r.usage = append(r.usage, event)
	return nil
}
func (r *usageAuditRecorder) RecordAudit(_ context.Context, event conversationstore.AuditEvent) error {
	r.audit = append(r.audit, event)
	return nil
}

func TestUsageAuditRegistrationRecordsMetadataOnly(t *testing.T) {
	recorder := &usageAuditRecorder{}
	manager := NewManager(nil)
	if err := manager.Register(NewUsageAuditRegistration(recorder, nil)); err != nil {
		t.Fatal(err)
	}
	ctx := security.WithAccessContext(context.Background(), security.MustAccessContext(security.Subject{TenantID: "tenant", UserID: "user", AuthMethod: security.AuthMethodJWT}))
	manager.Emit(ctx, State{Event: EventModelCallEnd, SessionID: "session", InteractionID: "interaction", CallID: "call", Model: "model", Input: "secret prompt", Output: "secret answer", InputTokens: 2, OutputTokens: 3, TotalTokens: 5, UsagePresent: true, EndedAt: time.Now()})
	if len(recorder.audit) != 1 || len(recorder.usage) != 1 {
		t.Fatalf("audit/usage = %d/%d", len(recorder.audit), len(recorder.usage))
	}
	if recorder.usage[0].TotalTokens == nil || *recorder.usage[0].TotalTokens != 5 {
		t.Fatalf("usage = %+v", recorder.usage[0])
	}
	if recorder.audit[0].Status != "ok" || recorder.audit[0].CallID != "call" || recorder.audit[0].CallID != recorder.usage[0].CallID {
		t.Fatalf("audit = %+v", recorder.audit[0])
	}
}

func TestAuditCallIdentifiersSurviveWithoutUsage(t *testing.T) {
	recorder := &usageAuditRecorder{}
	manager := NewManager(nil)
	if err := manager.Register(NewUsageAuditRegistration(recorder, nil, &UsageAuditOptions{AuditEvents: true})); err != nil {
		t.Fatal(err)
	}
	ctx := security.WithAccessContext(context.Background(), security.MustAccessContext(security.Subject{TenantID: "tenant", UserID: "user", AuthMethod: security.AuthMethodJWT}))
	states := []State{
		{Event: EventInteractionStart},
		{Event: EventModelCallEnd, CallID: "model-1"},
		{Event: EventModelCallEnd, CallID: "model-2", Error: "failed", ErrorType: "provider_error"},
		{Event: EventToolCallEnd, ToolName: "tool", ToolCallID: "tool-1"},
		{Event: EventToolCallEnd, ToolName: "tool", ToolCallID: "tool-2", Error: "failed", ErrorType: "tool_error"},
	}
	for _, state := range states {
		state.InteractionID = "interaction"
		manager.Emit(ctx, state)
	}
	if len(recorder.usage) != 0 || len(recorder.audit) != len(states) {
		t.Fatalf("usage/audit = %d/%d", len(recorder.usage), len(recorder.audit))
	}
	for i, audit := range recorder.audit {
		if audit.InteractionID != "interaction" || audit.CallID != states[i].CallID || audit.ToolCallID != states[i].ToolCallID {
			t.Fatalf("audit %d lost call identity: %+v", i, audit)
		}
	}
}

func TestUsageAuditRegistrationFeatureFlagsAndFailureDeduplication(t *testing.T) {
	recorder := &usageAuditRecorder{}
	options := &UsageAuditOptions{UsagePersistence: true, AuditEvents: false, SchemaVersion: "v1"}
	manager := NewManager(nil)
	if err := manager.Register(NewUsageAuditRegistration(recorder, nil, options)); err != nil {
		t.Fatal(err)
	}
	ctx := security.WithAccessContext(context.Background(), security.MustAccessContext(security.Subject{TenantID: "tenant", UserID: "user", AuthMethod: security.AuthMethodJWT}))
	state := State{SessionID: "session", InteractionID: "interaction", CallID: "call", Model: "model", Error: "failed", ErrorType: "provider_error", EndedAt: time.Now()}
	manager.Emit(ctx, State{Event: EventModelCallError, SessionID: state.SessionID, InteractionID: state.InteractionID, CallID: state.CallID, Model: state.Model, Error: state.Error, ErrorType: state.ErrorType, EndedAt: state.EndedAt})
	manager.Emit(ctx, State{Event: EventModelCallEnd, SessionID: state.SessionID, InteractionID: state.InteractionID, CallID: state.CallID, Model: state.Model, Error: state.Error, ErrorType: state.ErrorType, EndedAt: state.EndedAt})
	if len(recorder.audit) != 0 {
		t.Fatalf("audit flag disabled, got %d records", len(recorder.audit))
	}
	if len(recorder.usage) != 1 {
		t.Fatalf("failed call usage records = %d, want one", len(recorder.usage))
	}
	if recorder.usage[0].InputTokens != nil || recorder.usage[0].AmountSource != conversationstore.UsageAmountUnknown {
		t.Fatalf("missing usage was treated as present: %+v", recorder.usage[0])
	}
}

func TestUsageAuditFeatureUpdatesPreserveExplicitVersions(t *testing.T) {
	recorder := &usageAuditRecorder{}
	component := NewComponent(Spec{Hooks: []HookSpec{}})
	component.usageOptions = UsageAuditOptions{SchemaVersion: "schema-custom", PolicyVersion: "policy-custom", ComponentVersion: "recorder-custom"}
	if err := component.SetUsageAuditStore(recorder); err != nil {
		t.Fatal(err)
	}
	if err := component.Init(runtime.NewRuntime()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = component.Stop() })
	ctx := security.WithAccessContext(context.Background(), security.MustAccessContext(security.Subject{TenantID: "tenant", UserID: "user", AuthMethod: security.AuthMethodJWT}))
	component.GetManager().Emit(ctx, State{Event: EventModelCallEnd})
	if len(recorder.audit) != 0 || len(recorder.usage) != 0 {
		t.Fatal("explicit all-false switches were ignored")
	}
	component.SetUsageAuditFeatures(true, true)
	component.GetManager().Emit(ctx, State{Event: EventModelCallEnd, Error: "error", ErrorType: "provider_error", FallbackReason: "fallback"})
	if len(recorder.audit) != 1 || len(recorder.usage) != 1 {
		t.Fatal("feature update did not reach registered recorder")
	}
	audit, usage := recorder.audit[0], recorder.usage[0]
	if audit.SchemaVersion != "schema-custom" || audit.PolicyVersion != "policy-custom" || audit.ComponentVersion != "recorder-custom" || usage.SchemaVersion != "schema-custom" || usage.PolicyVersion != "policy-custom" {
		t.Fatalf("feature update replaced versions: %+v, %+v", audit, usage)
	}
	if audit.ReasonCode != "fallback" || usage.ReasonCode != "fallback" {
		t.Fatal("fallback must take precedence over generic error classification")
	}
}

type deadlineCheckingRecorder struct {
	t     *testing.T
	calls int
}

func (r *deadlineCheckingRecorder) check(ctx context.Context) error {
	r.t.Helper()
	r.calls++
	if err := ctx.Err(); err != nil {
		r.t.Errorf("persistence context canceled: %v", err)
	}
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > usageAuditPersistenceTimeout {
		r.t.Error("persistence deadline is missing or unbounded")
	}
	if _, err := security.AccessContextFromContext(ctx); err != nil {
		r.t.Error("persistence lost trusted identity")
	}
	return nil
}

func (r *deadlineCheckingRecorder) RecordUsage(ctx context.Context, _ conversationstore.UsageEvent) error {
	return r.check(ctx)
}
func (r *deadlineCheckingRecorder) RecordAudit(ctx context.Context, _ conversationstore.AuditEvent) error {
	return r.check(ctx)
}

func TestUsageAuditPersistenceRetainsIdentityWithIndependentDeadline(t *testing.T) {
	recorder := &deadlineCheckingRecorder{t: t}
	ctx, cancel := context.WithCancel(security.WithAccessContext(context.Background(), security.MustAccessContext(security.Subject{TenantID: "tenant", UserID: "user", AuthMethod: security.AuthMethodJWT})))
	cancel()
	registration := NewUsageAuditRegistration(recorder, nil, &UsageAuditOptions{UsagePersistence: true, AuditEvents: true})
	if returned := registration.Hook(ctx, State{Event: EventModelCallEnd}); returned != ctx {
		t.Fatal("recorder changed the caller context")
	}
	if recorder.calls != 2 {
		t.Fatalf("writes = %d, want 2", recorder.calls)
	}
}

func TestUsageAuditPersistsOnlyDurableBoundaries(t *testing.T) {
	wantAudit := map[Event]bool{
		EventInteractionStart: true, EventInteractionEnd: true,
		EventInteractionError: true, EventInteractionCancel: true,
		EventInteractionDegrade: true, EventModelCallEnd: true, EventToolCallEnd: true,
	}
	ctx := security.WithAccessContext(context.Background(), security.MustAccessContext(security.Subject{TenantID: "tenant", UserID: "user", AuthMethod: security.AuthMethodJWT}))
	for _, event := range AllEvents() {
		t.Run(string(event), func(t *testing.T) {
			recorder := &usageAuditRecorder{}
			manager := NewManager(nil)
			if err := manager.Register(NewUsageAuditRegistration(recorder, nil)); err != nil {
				t.Fatal(err)
			}
			traceCalls := 0
			if err := manager.Register(Registration{Events: AllEvents(), Tools: []string{AllTools}, Hook: func(ctx context.Context, state State) context.Context { traceCalls++; return ctx }}); err != nil {
				t.Fatal(err)
			}
			manager.Emit(ctx, State{Event: event, ToolName: "tool", Error: "private failure", ErrorType: "provider_error"})
			want := 0
			if wantAudit[event] {
				want = 1
			}
			if len(recorder.audit) != want {
				t.Fatalf("audit writes = %d, want %d", len(recorder.audit), want)
			}
			if want == 1 && (recorder.audit[0].Status != "error" || recorder.audit[0].ReasonCode != "provider_error") {
				t.Fatalf("failure audit = %+v", recorder.audit[0])
			}
			usage := 0
			if event == EventModelCallEnd {
				usage = 1
			}
			if len(recorder.usage) != usage {
				t.Fatalf("usage writes = %d, want %d", len(recorder.usage), usage)
			}
			if traceCalls != 1 {
				t.Fatal("audit filtering suppressed a tracing event")
			}
		})
	}
}

func TestUsageAuditSingleModelInteractionWriteCounts(t *testing.T) {
	for _, outcome := range []struct {
		name, errorType string
		exceptional     Event
	}{
		{name: "success"},
		{name: "model error", errorType: "provider_error", exceptional: EventInteractionError},
		{name: "timeout", errorType: "deadline_exceeded", exceptional: EventInteractionError},
		{name: "cancel", errorType: "canceled", exceptional: EventInteractionCancel},
		{name: "panic", errorType: "panic", exceptional: EventInteractionError},
	} {
		t.Run(outcome.name, func(t *testing.T) {
			recorder := &usageAuditRecorder{}
			manager := NewManager(nil)
			if err := manager.Register(NewUsageAuditRegistration(recorder, nil)); err != nil {
				t.Fatal(err)
			}
			ctx := security.WithAccessContext(context.Background(), security.MustAccessContext(security.Subject{TenantID: "tenant", UserID: "user", AuthMethod: security.AuthMethodJWT}))
			if outcome.name == "cancel" {
				canceled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = canceled
			}
			for _, event := range []Event{EventInteractionStart, EventIterationStart, EventStageStart, EventModelCallStart} {
				manager.Emit(ctx, State{Event: event})
			}
			state := State{Event: EventModelCallEnd, CallID: "call"}
			if outcome.errorType != "" {
				state.Error, state.ErrorType = "private failure", outcome.errorType
				errorState := state
				errorState.Event = EventModelCallError
				manager.Emit(ctx, errorState)
			}
			manager.Emit(ctx, state)
			for _, event := range []Event{EventStageEnd, EventIterationEnd, outcome.exceptional, EventInteractionEnd} {
				if event != "" {
					state.Event = event
					manager.Emit(ctx, state)
				}
			}
			want := 3
			if outcome.exceptional != "" {
				want++
			}
			if len(recorder.audit) != want || len(recorder.usage) != 1 {
				t.Fatalf("audit/usage = %d/%d, want %d/1", len(recorder.audit), len(recorder.usage), want)
			}
			if recorder.usage[0].CallID != "call" || recorder.usage[0].ReasonCode != outcome.errorType {
				t.Fatalf("usage = %+v", recorder.usage[0])
			}
		})
	}
}
