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
	"net/http"
	"strings"
)

const (
	AuthMethodStatic = "static"
	AuthMethodSystem = "system"
	AuthMethodJWT    = "jwt"

	ScopeSessionAdmin = "ai:sessions:admin"
)

var (
	ErrAuthenticationRequired = errors.New("authentication required")
	ErrResourceNotAccessible  = errors.New("resource not accessible")
)

// Subject is the authenticated identity supplied by a trusted server-side
// authenticator. Request payloads and ordinary client headers are not identity
// sources.
type Subject struct {
	TenantID   string
	UserID     string
	Roles      []string
	Scopes     []string
	AuthMethod string
}

func (s Subject) Validate() error {
	if strings.TrimSpace(s.TenantID) == "" || strings.TrimSpace(s.UserID) == "" || strings.TrimSpace(s.AuthMethod) == "" {
		return ErrAuthenticationRequired
	}
	return nil
}

func (s Subject) HasScope(scope string) bool {
	for _, candidate := range s.Scopes {
		if candidate == scope {
			return true
		}
	}
	return false
}

// ResourceReference is an untrusted resource requested by a caller. Its
// presence in AccessContext is not proof that access has been authorized.
type ResourceReference struct {
	Kind string
	ID   string
}

// AccessContext is passed explicitly to every Store operation that can read or
// mutate user-owned data.
type AccessContext struct {
	Subject            Subject
	ResourceReferences []ResourceReference
	systemCapability   bool
}

func NewAccessContext(subject Subject, references ...ResourceReference) (AccessContext, error) {
	if err := subject.Validate(); err != nil {
		return AccessContext{}, err
	}
	subject.Roles = append([]string(nil), subject.Roles...)
	subject.Scopes = append([]string(nil), subject.Scopes...)
	return AccessContext{Subject: subject, ResourceReferences: append([]ResourceReference(nil), references...)}, nil
}

func MustAccessContext(subject Subject, references ...ResourceReference) AccessContext {
	access, err := NewAccessContext(subject, references...)
	if err != nil {
		panic(err)
	}
	return access
}

func SystemAccessContext() AccessContext {
	return AccessContext{
		Subject:          Subject{TenantID: "system", UserID: "system", AuthMethod: AuthMethodSystem},
		systemCapability: true,
	}
}

func (a AccessContext) Validate() error { return a.Subject.Validate() }

func (a AccessContext) IsSystem() bool { return a.systemCapability }

// ValidateBusiness rejects the system capability from ordinary business
// operations. Maintenance operations are the only callers that may use it.
func (a AccessContext) ValidateBusiness() error {
	if err := a.Validate(); err != nil {
		return err
	}
	if a.IsSystem() {
		return ErrResourceNotAccessible
	}
	return nil
}

// CanAccessOwner intentionally grants no cross-tenant access. The reserved
// admin scope permits managing other owners only within the same tenant.
func (a AccessContext) CanAccessOwner(tenantID, ownerID string) bool {
	if a.Validate() != nil {
		return false
	}
	if a.IsSystem() {
		return true
	}
	if tenantID == "" || ownerID == "" {
		return false
	}
	if a.Subject.TenantID != tenantID {
		return false
	}
	return a.Subject.UserID == ownerID || a.Subject.HasScope(ScopeSessionAdmin)
}

// PolicyEnforcer centralizes ownership decisions used by API services and
// stores. Stores remain the final enforcement point.
type PolicyEnforcer interface {
	AuthorizeOwner(AccessContext, string, string) error
}

type OwnershipPolicy struct{}

func (OwnershipPolicy) AuthorizeOwner(access AccessContext, tenantID, ownerID string) error {
	if access.Validate() != nil {
		return ErrAuthenticationRequired
	}
	if !access.CanAccessOwner(tenantID, ownerID) {
		return ErrResourceNotAccessible
	}
	return nil
}

type accessContextKey struct{}

func WithAccessContext(ctx context.Context, access AccessContext) context.Context {
	return context.WithValue(ctx, accessContextKey{}, access)
}

func AccessContextFromContext(ctx context.Context) (AccessContext, error) {
	if ctx == nil {
		return AccessContext{}, ErrAuthenticationRequired
	}
	access, ok := ctx.Value(accessContextKey{}).(AccessContext)
	if !ok || access.Validate() != nil {
		return AccessContext{}, ErrAuthenticationRequired
	}
	return access, nil
}

// Authenticator derives a trusted identity for one HTTP request.
type Authenticator interface {
	Authenticate(*http.Request) (AccessContext, error)
}

// StaticAuthenticator supports deployments where identity is fixed by server
// configuration. It deliberately ignores request headers and bodies.
type StaticAuthenticator struct {
	access AccessContext
}

func NewStaticAuthenticator(subject Subject) (*StaticAuthenticator, error) {
	access, err := NewAccessContext(subject)
	if err != nil {
		return nil, err
	}
	return &StaticAuthenticator{access: access}, nil
}

func (a *StaticAuthenticator) Authenticate(_ *http.Request) (AccessContext, error) {
	if a == nil || a.access.Validate() != nil {
		return AccessContext{}, ErrAuthenticationRequired
	}
	return NewAccessContext(a.access.Subject, a.access.ResourceReferences...)
}
