/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0.
 */

package store

import "time"

const (
	derivedArtifactRetryBase = time.Minute
	derivedArtifactRetryMax  = time.Hour
)

// DerivedArtifactRetryAt returns a bounded exponential retry time. Attempt is
// one-based so the first cleanup failure waits for the base interval.
func DerivedArtifactRetryAt(failedAt time.Time, attempt uint32) time.Time {
	if attempt == 0 {
		attempt = 1
	}
	delay := derivedArtifactRetryBase
	for current := uint32(1); current < attempt && delay < derivedArtifactRetryMax; current++ {
		delay *= 2
		if delay > derivedArtifactRetryMax {
			delay = derivedArtifactRetryMax
		}
	}
	return failedAt.Add(delay)
}

type QuotaPolicy struct {
	MaxSessionsPerTenant int
	MaxTurnsPerSession   int
	MaxEventsPerTurn     int
	MaxSessionBytes      int64
	MaxTenantBytes       int64
	// MaxInMemoryEntities bounds conversation entities retained by MemoryStore.
	// Persistent backends use the session, turn/event, and byte quotas above.
	MaxInMemoryEntities int
}

func DefaultQuotaPolicy() QuotaPolicy {
	return QuotaPolicy{
		MaxSessionsPerTenant: 1000,
		MaxTurnsPerSession:   100,
		MaxEventsPerTurn:     1000,
		MaxSessionBytes:      10 << 20,
		MaxTenantBytes:       1 << 30,
		MaxInMemoryEntities:  100000,
	}
}

// WithDefaults fills omitted limits without replacing explicitly configured
// values. This keeps direct Store construction consistent with component
// configuration decoding.
func (p QuotaPolicy) WithDefaults() QuotaPolicy {
	defaults := DefaultQuotaPolicy()
	if p.MaxSessionsPerTenant <= 0 {
		p.MaxSessionsPerTenant = defaults.MaxSessionsPerTenant
	}
	if p.MaxTurnsPerSession <= 0 {
		p.MaxTurnsPerSession = defaults.MaxTurnsPerSession
	}
	if p.MaxEventsPerTurn <= 0 {
		p.MaxEventsPerTurn = defaults.MaxEventsPerTurn
	}
	if p.MaxSessionBytes <= 0 {
		p.MaxSessionBytes = defaults.MaxSessionBytes
	}
	if p.MaxTenantBytes <= 0 {
		p.MaxTenantBytes = defaults.MaxTenantBytes
	}
	if p.MaxInMemoryEntities <= 0 {
		p.MaxInMemoryEntities = defaults.MaxInMemoryEntities
	}
	return p
}

type LifecyclePolicy struct {
	Retention        time.Duration
	CleanupInterval  time.Duration
	CleanupBatchSize int
	InteractionLease time.Duration
}

type UsageAuditPolicy struct {
	Retention        time.Duration
	MaxUsageEvents   int
	MaxAuditEvents   int
	CleanupBatchSize int
}

// UsageAuditCleanupCutoff never permits cleanup inside the retention window.
func UsageAuditCleanupCutoff(cutoff, now time.Time, policy UsageAuditPolicy) time.Time {
	oldestAllowed := now.Add(-policy.WithDefaults().Retention)
	if cutoff.After(oldestAllowed) {
		return oldestAllowed
	}
	return cutoff
}

func DefaultUsageAuditPolicy() UsageAuditPolicy {
	return UsageAuditPolicy{Retention: 30 * 24 * time.Hour, MaxUsageEvents: 100000, MaxAuditEvents: 100000, CleanupBatchSize: 1000}
}

func (p UsageAuditPolicy) WithDefaults() UsageAuditPolicy {
	d := DefaultUsageAuditPolicy()
	if p.Retention <= 0 {
		p.Retention = d.Retention
	}
	if p.MaxUsageEvents <= 0 {
		p.MaxUsageEvents = d.MaxUsageEvents
	}
	if p.MaxAuditEvents <= 0 {
		p.MaxAuditEvents = d.MaxAuditEvents
	}
	if p.CleanupBatchSize <= 0 {
		p.CleanupBatchSize = d.CleanupBatchSize
	}
	return p
}

func DefaultLifecyclePolicy() LifecyclePolicy {
	return LifecyclePolicy{
		Retention: 24 * time.Hour, CleanupInterval: time.Hour,
		CleanupBatchSize: 100, InteractionLease: 30 * time.Minute,
	}
}

// WithDefaults fills omitted lifecycle settings independently so callers may
// override one policy value without having to copy every default.
func (p LifecyclePolicy) WithDefaults() LifecyclePolicy {
	defaults := DefaultLifecyclePolicy()
	if p.Retention <= 0 {
		p.Retention = defaults.Retention
	}
	if p.CleanupInterval <= 0 {
		p.CleanupInterval = defaults.CleanupInterval
	}
	if p.CleanupBatchSize <= 0 {
		p.CleanupBatchSize = defaults.CleanupBatchSize
	}
	if p.InteractionLease <= 0 {
		p.InteractionLease = defaults.InteractionLease
	}
	return p
}
