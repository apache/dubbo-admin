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

package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"dubbo-admin-ai/runtime"
	"dubbo-admin-ai/security"
	conversationstore "dubbo-admin-ai/store"
	gormstore "dubbo-admin-ai/store/gorm"
	memorystore "dubbo-admin-ai/store/memory"
	"gorm.io/gorm"
)

// MemoryComponent owns the single conversation Store shared by runtime
// consumers. The legacy HistoryMemory accessors remain for compatibility, but
// are initialized only when explicitly requested and are not a production data
// source.
type MemoryComponent struct {
	instanceName          string
	spec                  MemorySpec
	historyKey            HistoryKey
	maxTurns              int
	legacyOnce            sync.Once
	legacyErr             error
	memoryCtx             context.Context
	memory                *HistoryMemory
	store                 conversationstore.BackendStore
	gormStore             *gormstore.GormStore
	rt                    *runtime.Runtime
	cleanupCtx            context.Context
	cleanupCancel         context.CancelFunc
	cleanupWG             sync.WaitGroup
	cleanupMu             sync.RWMutex
	cleanupHandlers       map[string]conversationstore.DerivedArtifactCleanupHandler
	cleanupSessionCursor  conversationstore.CleanupCursor
	cleanupArtifactCursor conversationstore.CleanupCursor
	usageAuditPolicy      conversationstore.UsageAuditPolicy
}

func NewMemoryComponent(historyKey HistoryKey, maxTurns ...int) (runtime.Component, error) {
	limit := 100
	if len(maxTurns) > 0 {
		limit = maxTurns[0]
	}
	spec := MemorySpec{Backend: DefaultBackend, HistoryKey: historyKey, MaxTurns: limit}
	spec.applyPolicyDefaults()
	return &MemoryComponent{
		spec:            spec,
		historyKey:      historyKey,
		maxTurns:        limit,
		cleanupHandlers: make(map[string]conversationstore.DerivedArtifactCleanupHandler),
	}, nil
}

func (m *MemoryComponent) GetLifecyclePolicy() conversationstore.LifecyclePolicy {
	return conversationstore.LifecyclePolicy{
		Retention:        time.Duration(m.spec.Lifecycle.RetentionSeconds) * time.Second,
		CleanupInterval:  time.Duration(m.spec.Lifecycle.CleanupIntervalSeconds) * time.Second,
		CleanupBatchSize: m.spec.Lifecycle.CleanupBatchSize,
		InteractionLease: time.Duration(m.spec.Lifecycle.InteractionLeaseSeconds) * time.Second,
	}
}

// NewMemoryComponentFromSpec creates a memory component from the complete
// configuration decoded by the runtime factory.
func NewMemoryComponentFromSpec(spec MemorySpec) (runtime.Component, error) {
	spec.Backend = strings.ToLower(strings.TrimSpace(spec.Backend))
	if spec.Backend == "" {
		spec.Backend = DefaultBackend
	}
	if spec.HistoryKey == "" {
		spec.HistoryKey = ChatHistoryKey
	}
	if spec.MaxTurns == 0 {
		spec.MaxTurns = DefaultMemorySpec().MaxTurns
	}
	spec.applyPolicyDefaults()
	if spec.Database != nil {
		spec.Database.applyDefaults()
	}
	return &MemoryComponent{
		spec:            spec,
		historyKey:      spec.HistoryKey,
		maxTurns:        spec.MaxTurns,
		cleanupHandlers: make(map[string]conversationstore.DerivedArtifactCleanupHandler),
	}, nil
}

func (m *MemoryComponent) RegisterDerivedArtifactCleanupHandler(kind string, handler conversationstore.DerivedArtifactCleanupHandler) error {
	kind = strings.TrimSpace(kind)
	if kind == "" || handler == nil {
		return fmt.Errorf("derived artifact kind and cleanup handler are required")
	}
	m.cleanupMu.Lock()
	defer m.cleanupMu.Unlock()
	if _, exists := m.cleanupHandlers[kind]; exists {
		return fmt.Errorf("derived artifact cleanup handler %q is already registered", kind)
	}
	m.cleanupHandlers[kind] = handler
	return nil
}

func (m *MemoryComponent) Name() string {
	if m.instanceName != "" {
		return m.instanceName
	}
	return "memory"
}

func (m *MemoryComponent) SetName(name string) {
	m.instanceName = name
}

func (m *MemoryComponent) Validate() error {
	return m.spec.Validate()
}

func (m *MemoryComponent) Init(rt *runtime.Runtime) error {
	m.rt = rt
	quota := conversationstore.QuotaPolicy{
		MaxSessionsPerTenant: m.spec.Quota.MaxSessionsPerTenant, MaxTurnsPerSession: m.spec.MaxTurns,
		MaxEventsPerTurn: m.spec.Quota.MaxEventsPerTurn, MaxSessionBytes: m.spec.Quota.MaxSessionBytes,
		MaxTenantBytes: m.spec.Quota.MaxTenantBytes, MaxInMemoryEntities: m.spec.Quota.MaxInMemoryEntities,
	}
	lifecycle := m.GetLifecyclePolicy()
	usageAudit := conversationstore.UsageAuditPolicy{Retention: time.Duration(m.spec.UsageAudit.RetentionSeconds) * time.Second, MaxUsageEvents: m.spec.UsageAudit.MaxUsageEvents, MaxAuditEvents: m.spec.UsageAudit.MaxAuditEvents, CleanupBatchSize: m.spec.UsageAudit.CleanupBatchSize}
	m.usageAuditPolicy = usageAudit
	backend := m.spec.Backend
	if backend == "" {
		backend = DefaultBackend
	}
	switch backend {
	case "memory":
		m.store = memorystore.NewMemoryStoreWithPolicies(quota, lifecycle)
		m.store.(*memorystore.MemoryStore).SetUsageAuditPolicy(usageAudit)
	case "gorm":
		if m.spec.Database == nil {
			return fmt.Errorf("database is required for gorm backend")
		}
		db, err := gormstore.Open(m.spec.Database.Driver, m.spec.Database.DSN, &gorm.Config{})
		if err != nil {
			return fmt.Errorf("failed to open conversation database: %w", err)
		}
		sqlDB, err := db.DB()
		if err != nil {
			if closeDB, closeErr := db.DB(); closeErr == nil {
				_ = closeDB.Close()
			}
			return fmt.Errorf("failed to access conversation database: %w", err)
		}
		sqlDB.SetMaxOpenConns(m.spec.Database.MaxOpenConns)
		sqlDB.SetMaxIdleConns(m.spec.Database.MaxIdleConns)
		gormStore, err := gormstore.NewGormStoreWithPolicies(db, quota, lifecycle)
		if err != nil {
			_ = sqlDB.Close()
			return fmt.Errorf("failed to create gorm conversation store: %w", err)
		}
		if err := gormStore.Migrate(context.Background()); err != nil {
			_ = gormStore.Close()
			return fmt.Errorf("failed to migrate conversation database: %w", err)
		}
		gormStore.SetUsageAuditPolicy(usageAudit)
		m.gormStore = gormStore
		m.store = gormStore
	default:
		return fmt.Errorf("unsupported memory backend %q", backend)
	}
	rt.GetLogger().Info("Memory component initialized",
		"history_key", m.historyKey,
		"retention", time.Duration(m.spec.Lifecycle.RetentionSeconds)*time.Second)

	return nil
}

func (m *MemoryComponent) Start() error {
	if m.store == nil {
		return fmt.Errorf("store not initialized")
	}
	m.cleanupCtx, m.cleanupCancel = context.WithCancel(context.Background())
	m.cleanupWG.Add(1)
	go m.runCleanup()
	return nil
}

func (m *MemoryComponent) Stop() error {
	if m.cleanupCancel != nil {
		m.cleanupCancel()
	}
	m.cleanupWG.Wait()
	if m.gormStore != nil {
		err := m.gormStore.Close()
		m.gormStore = nil
		return err
	}
	return nil
}

func (m *MemoryComponent) runCleanup() {
	defer m.cleanupWG.Done()
	policy := m.GetLifecyclePolicy()
	ticker := time.NewTicker(policy.CleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			count, err := m.runCleanupOnce(m.cleanupCtx, now, policy.CleanupBatchSize)
			if err != nil {
				m.rt.GetLogger().Error("Failed to run conversation cleanup", "error", security.SafeError(err))
			} else if count > 0 {
				m.rt.GetLogger().Info("Completed conversation cleanup", "count", count)
			}
		case <-m.cleanupCtx.Done():
			return
		}
	}
}

func (m *MemoryComponent) runCleanupOnce(ctx context.Context, now time.Time, batchSize int) (int, error) {
	access := security.SystemAccessContext()
	usageAuditStore, ok := m.store.(conversationstore.UsageAuditMaintenanceStore)
	if !ok {
		return 0, fmt.Errorf("store does not support usage and audit cleanup")
	}
	usageCutoff := now.Add(-m.usageAuditPolicy.Retention)
	usageDeleted, err := usageAuditStore.CleanupUsage(ctx, access, usageCutoff, m.usageAuditPolicy.CleanupBatchSize)
	if err != nil {
		return usageDeleted, err
	}
	auditDeleted, err := usageAuditStore.CleanupAudit(ctx, access, usageCutoff, m.usageAuditPolicy.CleanupBatchSize)
	if err != nil {
		return usageDeleted, err
	}
	deleted, nextSession, err := m.store.DeleteExpiredPage(ctx, access, now, m.cleanupSessionCursor, batchSize)
	if err != nil {
		return deleted, err
	}
	m.cleanupSessionCursor = nextSession
	if !nextSession.Valid {
		m.cleanupSessionCursor = conversationstore.CleanupCursor{}
	}
	artifacts, nextArtifact, err := m.store.ListDerivedArtifactsForCleanupPage(ctx, access, now, m.cleanupArtifactCursor, batchSize)
	if err != nil {
		return deleted, err
	}
	m.cleanupArtifactCursor = nextArtifact
	if len(artifacts) == 0 {
		m.cleanupArtifactCursor = conversationstore.CleanupCursor{}
	}
	cleanedArtifact := false
	for i := range artifacts {
		m.cleanupMu.RLock()
		handler := m.cleanupHandlers[artifacts[i].Kind]
		m.cleanupMu.RUnlock()
		if handler == nil {
			if err := m.store.MarkDerivedArtifactCleanupFailed(ctx, access, artifacts[i].ID, "HANDLER_REQUIRED", now); err != nil {
				return deleted, err
			}
			continue
		}
		if err := handler.CleanupDerivedArtifact(ctx, artifacts[i]); err != nil {
			if markErr := m.store.MarkDerivedArtifactCleanupFailed(ctx, access, artifacts[i].ID, "CLEANUP_FAILED", now); markErr != nil {
				return deleted, markErr
			}
			m.rt.GetLogger().Error("Failed to clean derived artifact", "artifact_id", artifacts[i].ID, "artifact_kind", artifacts[i].Kind, "error", security.SafeError(err))
			continue
		}
		if err := m.store.MarkDerivedArtifactDeleted(ctx, access, artifacts[i].ID); err != nil {
			return deleted, err
		}
		cleanedArtifact = true
	}
	if !cleanedArtifact {
		return deleted + usageDeleted + auditDeleted, nil
	}
	finalized, nextSession, err := m.store.DeleteExpiredPage(ctx, access, now, m.cleanupSessionCursor, batchSize)
	m.cleanupSessionCursor = nextSession
	if !nextSession.Valid {
		m.cleanupSessionCursor = conversationstore.CleanupCursor{}
	}
	return deleted + finalized + usageDeleted + auditDeleted, err
}

// initLegacyMemory initializes the deprecated context-backed memory only when
// a legacy caller explicitly requests it.
func (m *MemoryComponent) initLegacyMemory() {
	m.legacyOnce.Do(func() {
		m.memoryCtx = NewMemoryContext(m.historyKey)
		m.memory, m.legacyErr = GetHistoryMemory(m.memoryCtx, m.historyKey)
	})
}

// GetContext returns the deprecated context-backed memory context.
func (m *MemoryComponent) GetContext() context.Context {
	m.initLegacyMemory()
	return m.memoryCtx
}

// GetStore returns the single conversation Store shared by runtime consumers.
func (m *MemoryComponent) GetStore() (conversationstore.Store, error) {
	if m.store == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	return m.store, nil
}

// GetMaintenanceStore returns the privileged lifecycle contract for runtime
// components that own cleanup or migration.
func (m *MemoryComponent) GetMaintenanceStore() (conversationstore.MaintenanceStore, error) {
	if m.store == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	return m.store, nil
}

// GetMemory returns the deprecated HistoryMemory instance for compatibility.
func (m *MemoryComponent) GetMemory() (*HistoryMemory, error) {
	m.initLegacyMemory()
	return m.memory, m.legacyErr
}
