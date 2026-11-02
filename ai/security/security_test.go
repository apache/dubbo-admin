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

package security

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
)

func TestStaticAuthenticatorIgnoresClientIdentityHeaders(t *testing.T) {
	authenticator, err := NewStaticAuthenticator(Subject{
		TenantID: "trusted-tenant", UserID: "trusted-user", AuthMethod: AuthMethodStatic,
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set("X-Tenant-ID", "attacker-tenant")
	request.Header.Set("X-User-ID", "attacker-user")

	access, err := authenticator.Authenticate(request)
	if err != nil {
		t.Fatal(err)
	}
	if access.Subject.TenantID != "trusted-tenant" || access.Subject.UserID != "trusted-user" {
		t.Fatalf("authenticated subject = %+v", access.Subject)
	}
}

func TestAccessContextRequiresAuthenticatedSubject(t *testing.T) {
	if _, err := AccessContextFromContext(context.Background()); !errors.Is(err, ErrAuthenticationRequired) {
		t.Fatalf("AccessContextFromContext() error = %v, want ErrAuthenticationRequired", err)
	}
	access := MustAccessContext(Subject{TenantID: "tenant", UserID: "user", AuthMethod: AuthMethodStatic})
	got, err := AccessContextFromContext(WithAccessContext(context.Background(), access))
	if err != nil || got.Subject.UserID != "user" {
		t.Fatalf("AccessContextFromContext() = %+v, %v", got, err)
	}
}

func TestOwnershipPolicyDoesNotGrantCrossTenantAdmin(t *testing.T) {
	policy := OwnershipPolicy{}
	admin := MustAccessContext(Subject{
		TenantID: "tenant-a", UserID: "admin", Scopes: []string{ScopeSessionAdmin}, AuthMethod: AuthMethodStatic,
	})
	if err := policy.AuthorizeOwner(admin, "tenant-a", "other-user"); err != nil {
		t.Fatalf("same-tenant admin denied: %v", err)
	}
	if err := policy.AuthorizeOwner(admin, "tenant-b", "other-user"); !errors.Is(err, ErrResourceNotAccessible) {
		t.Fatalf("cross-tenant admin error = %v, want ErrResourceNotAccessible", err)
	}
}

func TestSystemAuthMethodDoesNotGrantSystemCapability(t *testing.T) {
	forged := MustAccessContext(Subject{TenantID: "system", UserID: "system", AuthMethod: AuthMethodSystem})
	if forged.IsSystem() {
		t.Fatal("ordinary AccessContext with system auth method gained system capability")
	}
	if forged.CanAccessOwner("tenant-a", "owner-a") {
		t.Fatal("forged system subject bypassed ownership")
	}
	if !SystemAccessContext().IsSystem() {
		t.Fatal("SystemAccessContext() did not carry system capability")
	}
}
