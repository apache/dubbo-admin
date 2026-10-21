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

package store

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ValidateSession defines the model validation shared by every Store backend.
func ValidateSession(session *Session) error {
	if session == nil {
		return errors.New("session is nil")
	}
	if strings.TrimSpace(session.ID) == "" {
		return errors.New("session id is required")
	}
	if strings.TrimSpace(session.TenantID) == "" {
		return errors.New("session tenant_id is required")
	}
	if strings.TrimSpace(session.OwnerID) == "" {
		return errors.New("session owner_id is required")
	}
	if session.Version == 0 {
		return errors.New("session version is required")
	}
	if session.CreatedAt.IsZero() {
		return errors.New("session created_at is required")
	}
	if session.UpdatedAt.IsZero() {
		return errors.New("session updated_at is required")
	}
	if strings.TrimSpace(session.Status) == "" {
		return errors.New("session status is required")
	}
	return nil
}

// ValidateSessionActive applies the common lifecycle gate used before reading
// or mutating conversation history.
func ValidateSessionActive(sessionID, status string, expired bool) error {
	if status == SessionStatusDeletionPending || status == SessionStatusDerivedCleanup || status == SessionStatusDeleted {
		return ErrSessionNotFound
	}
	if status != SessionStatusActive {
		return fmt.Errorf("session %q is not active", sessionID)
	}
	if expired {
		return ErrSessionExpired
	}
	return nil
}

// IsSessionExpired applies the persisted expiration timestamp when present,
// falling back to the backend's legacy activity window otherwise.
func IsSessionExpired(expiresAt *time.Time, updatedAt, now time.Time, legacyWindow time.Duration) bool {
	if expiresAt != nil {
		return !expiresAt.After(now)
	}
	return now.Sub(updatedAt) > legacyWindow
}

func SessionIDFromCanonicalSource(canonicalSourceID string) (string, error) {
	const prefix = "session:"
	if !strings.HasPrefix(canonicalSourceID, prefix) || strings.TrimSpace(strings.TrimPrefix(canonicalSourceID, prefix)) == "" {
		return "", ErrCanonicalSourceNotWritable
	}
	return strings.TrimPrefix(canonicalSourceID, prefix), nil
}
