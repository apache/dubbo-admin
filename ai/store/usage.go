package store

import (
	"context"
	"time"

	"dubbo-admin-ai/security"
)

const (
	UsageCallInteraction = "interaction"
	UsageCallModel       = "model"
	UsageCallTool        = "tool"

	UsageAmountEstimated = "estimated"
	UsageAmountProvider  = "provider"
	UsageAmountUnknown   = "unknown"
	UsageCostKnown       = "known"
	UsageCostUnknown     = "unknown"
)

// UsageEvent contains privacy-safe accounting metadata. Token pointers retain
// the distinction between an unavailable value and a real zero.
type UsageEvent struct {
	EstimatedInputTokens *int
	TemplateRef          string
	CompilerVersion      string
	EstimatorVersion     string
	SchemaVersion        string
	ID                   string
	TenantID             string
	UserID               string
	SessionID            string
	InteractionID        string
	CallID               string
	CallType             string
	Provider             string
	Model                string
	ProviderVersion      string
	ModelVersion         string
	PolicyVersion        string
	AmountSource         string
	InputTokens          *int
	OutputTokens         *int
	TotalTokens          *int
	PricingVersion       string
	CostStatus           string
	Cost                 *float64
	Status               string
	ReasonCode           string
	Attempt              int
	Retry                bool
	Auxiliary            bool
	CreatedAt            time.Time
}

type AuditEvent struct {
	SchemaVersion    string
	ID               string
	TenantID         string
	UserID           string
	SessionID        string
	InteractionID    string
	CallID           string // Model call identifier, shared with UsageEvent.
	ToolCallID       string // Provider tool request reference, scoped to the interaction.
	EventType        string
	Status           string
	ReasonCode       string
	PolicyVersion    string
	ComponentVersion string
	CreatedAt        time.Time
}

type PricingRecord struct {
	Provider         string
	Model            string
	PricingVersion   string
	InputTokenPrice  float64
	OutputTokenPrice float64
	Currency         string
	EffectiveFrom    time.Time
	EffectiveUntil   *time.Time
}

type PricingResolver interface {
	Resolve(context.Context, string, string, time.Time) (PricingRecord, error)
}

type UsageAuditStore interface {
	RecordUsage(context.Context, UsageEvent) error
	RecordAudit(context.Context, AuditEvent) error
}

type UsageAuditMaintenanceStore interface {
	UsageAuditStore
	// Cleanup requires the system capability and clamps cutoff to the retention
	// boundary. limit is capped by CleanupBatchSize. Memory limits records scanned
	// per call and advances a cursor; callers must not treat zero removals as
	// proof that all expired records have been visited.
	CleanupUsage(context.Context, security.AccessContext, time.Time, int) (int, error)
	CleanupAudit(context.Context, security.AccessContext, time.Time, int) (int, error)
}
