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
	"fmt"
	"strings"
)

const (
	DefaultBackend      = "memory"
	DefaultMaxOpenConns = 100
	DefaultMaxIdleConns = 10
)

type MemorySpec struct {
	Backend    string        `yaml:"backend"`
	HistoryKey HistoryKey    `yaml:"history_key"` // History key name
	MaxTurns   int           `yaml:"max_turns"`   // Maximum conversation turns
	Database   *DatabaseSpec `yaml:"database"`
	Quota      QuotaSpec     `yaml:"quota"`
	Lifecycle  LifecycleSpec `yaml:"lifecycle"`
}

type QuotaSpec struct {
	MaxSessionsPerTenant int   `yaml:"max_sessions_per_tenant"`
	MaxEventsPerTurn     int   `yaml:"max_events_per_turn"`
	MaxSessionBytes      int64 `yaml:"max_session_bytes"`
	MaxTenantBytes       int64 `yaml:"max_tenant_bytes"`
	MaxInMemoryEntities  int   `yaml:"max_in_memory_entities"`
}

type LifecycleSpec struct {
	RetentionSeconds        int `yaml:"retention_seconds"`
	CleanupIntervalSeconds  int `yaml:"cleanup_interval_seconds"`
	CleanupBatchSize        int `yaml:"cleanup_batch_size"`
	InteractionLeaseSeconds int `yaml:"interaction_lease_seconds"`
}

type DatabaseSpec struct {
	Driver       string `yaml:"driver"`
	DSN          string `yaml:"dsn"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
}

func DefaultMemorySpec() *MemorySpec {
	spec := &MemorySpec{
		Backend:    DefaultBackend,
		HistoryKey: ChatHistoryKey,
		MaxTurns:   100,
	}
	spec.applyPolicyDefaults()
	return spec
}

func (s *MemorySpec) applyPolicyDefaults() {
	if s.Quota.MaxSessionsPerTenant == 0 {
		s.Quota.MaxSessionsPerTenant = 1000
	}
	if s.Quota.MaxEventsPerTurn == 0 {
		s.Quota.MaxEventsPerTurn = 1000
	}
	if s.Quota.MaxSessionBytes == 0 {
		s.Quota.MaxSessionBytes = 10 << 20
	}
	if s.Quota.MaxTenantBytes == 0 {
		s.Quota.MaxTenantBytes = 1 << 30
	}
	if s.Quota.MaxInMemoryEntities == 0 {
		s.Quota.MaxInMemoryEntities = 100000
	}
	if s.Lifecycle.RetentionSeconds == 0 {
		s.Lifecycle.RetentionSeconds = 86400
	}
	if s.Lifecycle.CleanupIntervalSeconds == 0 {
		s.Lifecycle.CleanupIntervalSeconds = 3600
	}
	if s.Lifecycle.CleanupBatchSize == 0 {
		s.Lifecycle.CleanupBatchSize = 100
	}
	if s.Lifecycle.InteractionLeaseSeconds == 0 {
		s.Lifecycle.InteractionLeaseSeconds = 1800
	}
}

func (s MemorySpec) Validate() error {
	s.applyPolicyDefaults()
	backend := strings.ToLower(strings.TrimSpace(s.Backend))
	if backend == "" {
		backend = DefaultBackend
	}
	if backend != "memory" && backend != "gorm" {
		return fmt.Errorf("unsupported memory backend %q", s.Backend)
	}
	if s.MaxTurns <= 0 {
		return fmt.Errorf("max_turns must be greater than 0")
	}
	if s.Quota.MaxSessionsPerTenant <= 0 || s.Quota.MaxEventsPerTurn <= 0 || s.Quota.MaxSessionBytes <= 0 || s.Quota.MaxTenantBytes <= 0 || s.Quota.MaxInMemoryEntities <= 0 {
		return fmt.Errorf("all quota limits must be greater than 0")
	}
	if s.Lifecycle.RetentionSeconds <= 0 || s.Lifecycle.CleanupIntervalSeconds <= 0 || s.Lifecycle.CleanupBatchSize <= 0 || s.Lifecycle.InteractionLeaseSeconds <= 0 {
		return fmt.Errorf("all lifecycle settings must be greater than 0")
	}
	if s.Database == nil {
		if backend == "gorm" {
			return fmt.Errorf("database is required for gorm backend")
		}
		return nil
	}
	if backend != "gorm" {
		return nil
	}

	database := *s.Database
	database.applyDefaults()
	if strings.ToLower(strings.TrimSpace(database.Driver)) != "mysql" && strings.ToLower(strings.TrimSpace(database.Driver)) != "postgres" {
		return fmt.Errorf("unsupported database driver %q", s.Database.Driver)
	}
	if strings.TrimSpace(database.DSN) == "" {
		return fmt.Errorf("database dsn is required")
	}
	if database.MaxOpenConns <= 0 {
		return fmt.Errorf("max_open_conns must be greater than 0")
	}
	if database.MaxIdleConns <= 0 {
		return fmt.Errorf("max_idle_conns must be greater than 0")
	}
	if database.MaxIdleConns > database.MaxOpenConns {
		return fmt.Errorf("max_idle_conns must not exceed max_open_conns")
	}
	return nil
}

func (d *DatabaseSpec) applyDefaults() {
	if d.MaxOpenConns == 0 {
		d.MaxOpenConns = DefaultMaxOpenConns
	}
	if d.MaxIdleConns == 0 {
		d.MaxIdleConns = DefaultMaxIdleConns
	}
}
