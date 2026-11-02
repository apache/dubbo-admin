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

package testutils

import (
	"context"
	"time"

	"dubbo-admin-ai/security"
	conversationstore "dubbo-admin-ai/store"
)

func TestAccess() security.AccessContext {
	return security.MustAccessContext(security.Subject{
		TenantID: "test-tenant", UserID: "test-user", Roles: []string{"user"}, AuthMethod: security.AuthMethodStatic,
	})
}

func ContextWithTestAccess(ctx context.Context) context.Context {
	return security.WithAccessContext(ctx, TestAccess())
}

func TestSession(id string, now time.Time) *conversationstore.Session {
	expiresAt := now.Add(24 * time.Hour)
	access := TestAccess()
	return &conversationstore.Session{
		ID: id, TenantID: access.Subject.TenantID, OwnerID: access.Subject.UserID, Version: 1,
		CreatedAt: now, UpdatedAt: now, ExpiresAt: &expiresAt, Status: "active",
	}
}
