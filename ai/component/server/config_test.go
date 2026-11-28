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

package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"dubbo-admin-ai/security"
	conversationstore "dubbo-admin-ai/store"
)

func TestAuthenticationSpecRoleDefaultsDistinguishMissingAndEmpty(t *testing.T) {
	missing := AuthenticationSpec{}
	missing.applyDefaults()
	if len(missing.Roles) != 1 || missing.Roles[0] != "user" {
		t.Fatalf("missing roles defaulted to %#v, want [user]", missing.Roles)
	}

	explicitEmpty := AuthenticationSpec{Roles: []string{}}
	explicitEmpty.applyDefaults()
	if explicitEmpty.Roles == nil || len(explicitEmpty.Roles) != 0 {
		t.Fatalf("explicit empty roles changed to %#v", explicitEmpty.Roles)
	}
}

func TestNormalizeHostIPv6(t *testing.T) {
	if got := normalizeHost(" [::1] "); got != "::1" {
		t.Fatalf("normalizeHost() = %q, want %q", got, "::1")
	}
	if !isLoopbackHost("[::1]") || !isLoopbackHost("::1") {
		t.Fatal("IPv6 loopback host was not recognized")
	}
}

type recordingMaintenanceStore struct {
	conversationstore.MaintenanceStore
	called   bool
	access   security.AccessContext
	tenantID string
	ownerID  string
	err      error
}

func (s *recordingMaintenanceStore) DeleteExpired(context.Context, security.AccessContext, time.Time, ...int) (int, error) {
	return 0, nil
}

func (s *recordingMaintenanceStore) AdoptLegacySessions(_ context.Context, access security.AccessContext, tenantID, ownerID string) (int, error) {
	s.called = true
	s.access = access
	s.tenantID = tenantID
	s.ownerID = ownerID
	return 0, s.err
}

func TestAdoptLegacySessionsWiring(t *testing.T) {
	subject := security.Subject{TenantID: "tenant-a", UserID: "user-a", AuthMethod: security.AuthMethodStatic}
	store := &recordingMaintenanceStore{}
	if err := adoptLegacySessions(context.Background(), store, true, subject); err != nil {
		t.Fatalf("adoptLegacySessions() error = %v", err)
	}
	if !store.called || !store.access.IsSystem() || store.tenantID != subject.TenantID || store.ownerID != subject.UserID {
		t.Fatalf("adoption call = called:%v system:%v tenant:%q owner:%q", store.called, store.access.IsSystem(), store.tenantID, store.ownerID)
	}
	store.called = false
	if err := adoptLegacySessions(context.Background(), store, false, subject); err != nil {
		t.Fatalf("disabled adoptLegacySessions() error = %v", err)
	}
	if store.called {
		t.Fatal("disabled adoption called maintenance store")
	}

	store.err = errors.New("adoption failed")
	if err := adoptLegacySessions(context.Background(), store, true, subject); !errors.Is(err, store.err) {
		t.Fatalf("adoptLegacySessions() error = %v, want %v", err, store.err)
	}
}
