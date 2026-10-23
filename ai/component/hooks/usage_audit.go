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
	if logger == nil {
		logger = slog.Default()
	}
	defaults := DefaultUsageAuditOptions()
	options := &defaults
	if len(optionPtrs) > 0 && optionPtrs[0] != nil {
		options = optionPtrs[0]
	}
	*options = options.withVersions()
	// Detailed iteration/stage and call-start events belong to tracing. Call-end
	// events already carry failures, including panic and cancellation outcomes.
	return Registration{Events: []Event{
		EventInteractionStart, EventInteractionEnd, EventInteractionError,
		EventInteractionCancel, EventInteractionDegrade,
		EventModelCallEnd, EventToolCallEnd,
	}, Tools: []string{AllTools}, Hook: func(ctx context.Context, state State) context.Context {
		access, err := security.AccessContextFromContext(ctx)
		if err != nil || store == nil {
			return ctx
		}
		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), usageAuditPersistenceTimeout)
		defer cancel()
		now := state.EndedAt
		if now.IsZero() {
			now = time.Now()
		}
		audit := conversationstore.AuditEvent{ID: uuid.NewString(), SchemaVersion: options.SchemaVersion, TenantID: access.Subject.TenantID, UserID: access.Subject.UserID, SessionID: state.SessionID, InteractionID: state.InteractionID, CallID: state.CallID, ToolCallID: state.ToolCallID, EventType: string(state.Event), Status: "ok", ReasonCode: state.ReasonCode(), PolicyVersion: options.PolicyVersion, ComponentVersion: options.ComponentVersion, CreatedAt: now}
		if state.Error != "" {
			audit.Status = "error"
		}
		if options.AuditEvents {
			if err := store.RecordAudit(persistCtx, audit); err != nil {
				logger.Warn("failed to record AI audit event", "event", state.Event, "error_type", ErrorType(err))
			}
		}
		if options.UsagePersistence && state.Event == EventModelCallEnd {
			usage := conversationstore.UsageEvent{ID: uuid.NewString(), SchemaVersion: options.SchemaVersion, TenantID: access.Subject.TenantID, UserID: access.Subject.UserID, SessionID: state.SessionID, InteractionID: state.InteractionID, CallID: state.CallID, CallType: conversationstore.UsageCallModel, Model: state.Model, PolicyVersion: options.PolicyVersion, AmountSource: conversationstore.UsageAmountUnknown, CostStatus: conversationstore.UsageCostUnknown, Status: audit.Status, ReasonCode: state.ReasonCode(), CreatedAt: now}
			if state.UsagePresent {
				usage.AmountSource = conversationstore.UsageAmountProvider
				usage.InputTokens = intPtr(state.InputTokens)
				usage.OutputTokens = intPtr(state.OutputTokens)
				usage.TotalTokens = intPtr(state.TotalTokens)
			}
			if state.EstimatorVersion != "" {
				usage.EstimatedInputTokens = intPtr(state.EstimatedInputTokens)
			}
			usage.TemplateRef = state.TemplateRef
			usage.CompilerVersion = state.CompilerVersion
			usage.EstimatorVersion = state.EstimatorVersion
			if err := store.RecordUsage(persistCtx, usage); err != nil {
				logger.Warn("failed to record AI usage event", "event", state.Event, "error_type", ErrorType(err))
			}
		}
		return ctx
	}}
}

func intPtr(value int) *int { return &value }

func (s State) ReasonCode() string {
	if s.FallbackReason != "" {
		return s.FallbackReason
	}
	return s.ErrorType
}
