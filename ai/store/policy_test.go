/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0.
 */

package store

import (
	"testing"
	"time"
)

func TestPoliciesWithDefaultsPreserveOverrides(t *testing.T) {
	quota := (QuotaPolicy{MaxSessionBytes: 123}).WithDefaults()
	if quota.MaxSessionBytes != 123 || quota.MaxSessionsPerTenant <= 0 || quota.MaxTurnsPerSession <= 0 || quota.MaxEventsPerTurn <= 0 || quota.MaxTenantBytes <= 0 || quota.MaxInMemoryEntities <= 0 {
		t.Fatalf("quota defaults = %#v", quota)
	}

	lifecycle := (LifecyclePolicy{Retention: 2 * time.Hour}).WithDefaults()
	if lifecycle.Retention != 2*time.Hour || lifecycle.CleanupInterval <= 0 || lifecycle.CleanupBatchSize <= 0 || lifecycle.InteractionLease <= 0 {
		t.Fatalf("lifecycle defaults = %#v", lifecycle)
	}
}
