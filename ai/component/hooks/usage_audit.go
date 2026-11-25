package hooks

import (
	"context"
	"log/slog"
	"time"

	"dubbo-admin-ai/security"
	conversationstore "dubbo-admin-ai/store"

	"github.com/google/uuid"
)

type UsageAuditOptions struct {
	UsagePersistence bool
	AuditEvents      bool
	SchemaVersion    string
	PolicyVersion    string
	ComponentVersion string
}

const (
	UsageAuditSchemaVersion      = "v1"
	UsageAuditPolicyVersion      = "subject-owned-metadata-v1"
	UsageAuditComponentVersion   = "usage-audit-recorder-v1"
	usageAuditPersistenceTimeout = 5 * time.Second
)

func DefaultUsageAuditOptions() UsageAuditOptions {
	return (UsageAuditOptions{UsagePersistence: true, AuditEvents: true}).withVersions()
}

func (o UsageAuditOptions) withVersions() UsageAuditOptions {
	if o.SchemaVersion == "" {
		o.SchemaVersion = UsageAuditSchemaVersion
	}
	if o.PolicyVersion == "" {
		o.PolicyVersion = UsageAuditPolicyVersion
	}
	if o.ComponentVersion == "" {
		o.ComponentVersion = UsageAuditComponentVersion
	}
	return o
}

// NewUsageAuditRegistration records only identifiers, statuses, reason codes,
// versions and token counts. It never captures variable model or tool content.
func NewUsageAuditRegistration(store conversationstore.UsageAuditStore, logger *slog.Logger, optionPtrs ...*UsageAuditOptions) Registration {
	options := DefaultUsageAuditOptions()
	if len(optionPtrs) > 0 && optionPtrs[0] != nil {
		options = optionPtrs[0].withVersions()
	}
	return newUsageAuditRegistration(store, logger, func() UsageAuditOptions { return options })
}

// The component supplies synchronized snapshots for runtime rollout switches.
func newUsageAuditRegistration(store conversationstore.UsageAuditStore, logger *slog.Logger, snapshot func() UsageAuditOptions) Registration {
	if logger == nil {
		logger = slog.Default()
	}
	return newUsageAuditRegistrationWithSink(store, logger, snapshot, func(ctx context.Context, record usageAuditRecord) {
		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), usageAuditPersistenceTimeout)
		defer cancel()
		record.persist(persistCtx, store, logger)
	})
}

// newAsyncUsageAuditRegistration hands records to the bounded writer so the
// model/tool hot path never waits for a Store write or its deadline.
func newAsyncUsageAuditRegistration(writer *usageAuditWriter, logger *slog.Logger, snapshot func() UsageAuditOptions) Registration {
	if logger == nil {
		logger = slog.Default()
	}
	return newUsageAuditRegistrationWithSink(writer.store, logger, snapshot, func(_ context.Context, record usageAuditRecord) {
		writer.Enqueue(record)
	})
}

func newUsageAuditRegistrationWithSink(store conversationstore.UsageAuditStore, logger *slog.Logger, snapshot func() UsageAuditOptions, sink func(context.Context, usageAuditRecord)) Registration {
	// Detailed iteration/stage and call-start events belong to tracing. Call-end
	// events already carry failures, including panic and cancellation outcomes.
	return Registration{Events: []Event{
		EventInteractionStart, EventInteractionEnd, EventInteractionError,
		EventInteractionCancel, EventInteractionDegrade,
		EventModelCallEnd, EventToolCallEnd,
	}, Tools: []string{AllTools}, Hook: func(ctx context.Context, state State) context.Context {
		if store == nil {
			return ctx
		}
		options := snapshot().withVersions()
		if !options.UsagePersistence && !options.AuditEvents {
			return ctx
		}
		access, err := security.AccessContextFromContext(ctx)
		if err != nil {
			logger.Warn("usage audit skipped: no access context", "event", state.Event)
			return ctx
		}
		record := newUsageAuditRecord(state, options, access)
		if record.audit == nil && record.usage == nil {
			return ctx
		}
		sink(ctx, record)
		return ctx
	}}
}

// newUsageAuditRecord builds only metadata: identifiers, status, reason code,
// versions and token counts. Prompt, response, and tool payloads are never
// copied into the record.
func newUsageAuditRecord(state State, options UsageAuditOptions, access security.AccessContext) usageAuditRecord {
	now := state.EndedAt
	if now.IsZero() {
		now = time.Now()
	}
	status := "ok"
	if state.Error != "" {
		status = "error"
	}
	record := usageAuditRecord{access: access, event: state.Event}
	if options.AuditEvents {
		record.audit = &conversationstore.AuditEvent{ID: uuid.NewString(), SchemaVersion: options.SchemaVersion, TenantID: access.Subject.TenantID, UserID: access.Subject.UserID, SessionID: state.SessionID, InteractionID: state.InteractionID, CallID: state.CallID, ToolCallID: state.ToolCallID, EventType: string(state.Event), Status: status, ReasonCode: state.ReasonCode(), PolicyVersion: options.PolicyVersion, ComponentVersion: options.ComponentVersion, CreatedAt: now}
	}
	if options.UsagePersistence && state.Event == EventModelCallEnd {
		usage := conversationstore.UsageEvent{ID: uuid.NewString(), SchemaVersion: options.SchemaVersion, TenantID: access.Subject.TenantID, UserID: access.Subject.UserID, SessionID: state.SessionID, InteractionID: state.InteractionID, CallID: state.CallID, CallType: conversationstore.UsageCallModel, Model: state.Model, PolicyVersion: options.PolicyVersion, AmountSource: conversationstore.UsageAmountUnknown, CostStatus: conversationstore.UsageCostUnknown, Status: status, ReasonCode: state.ReasonCode(), CreatedAt: now}
		if state.UsagePresent {
			usage.AmountSource = conversationstore.UsageAmountProvider
			usage.InputTokens = intPtr(state.InputTokens)
			usage.OutputTokens = intPtr(state.OutputTokens)
			usage.TotalTokens = intPtr(state.TotalTokens)
		}
		record.usage = &usage
	}
	return record
}

func intPtr(value int) *int { return &value }

func (s State) ReasonCode() string {
	if s.FallbackReason != "" {
		return s.FallbackReason
	}
	return s.ErrorType
}
