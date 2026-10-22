package config

// FeatureFlags controls optional Phase 1 accounting and rollout behavior.
// Security isolation and baseline sanitization deliberately have no flag.
type FeatureFlags struct {
	ProductionAuth   bool `yaml:"production_auth" json:"production_auth"`
	UsagePersistence bool `yaml:"usage_persistence" json:"usage_persistence"`
	AuditEvents      bool `yaml:"audit_events" json:"audit_events"`
}

func DefaultFeatureFlags() FeatureFlags {
	return FeatureFlags{ProductionAuth: false, UsagePersistence: true, AuditEvents: true}
}
