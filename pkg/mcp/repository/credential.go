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

package repository

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

type Credentials struct {
	resources   manager.ResourceManager
	conditional manager.ConditionalResourceManager
}

func NewCredentials(resources manager.ResourceManager) (*Credentials, error) {
	conditional, ok := resources.(manager.ConditionalResourceManager)
	if !ok {
		return nil, fmt.Errorf("resource manager does not support conditional mutations")
	}
	return &Credentials{resources: resources, conditional: conditional}, nil
}

func (r *Credentials) Create(mesh, serverID, displayName string, expiresAt time.Time) (*meshresource.MCPCredentialResource, string, error) {
	if mesh == "" || serverID == "" || strings.TrimSpace(displayName) == "" || !expiresAt.After(time.Now()) {
		return nil, "", fmt.Errorf("mesh, server, display name and a future expiration are required")
	}
	_, exists, err := r.resources.GetByKey(meshresource.MCPServerKind, coremodel.BuildResourceKey(mesh, serverID))
	if err != nil {
		return nil, "", err
	}
	if !exists {
		return nil, "", fmt.Errorf("MCP server %q not found", serverID)
	}
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, "", fmt.Errorf("generate credential secret: %w", err)
	}
	secret := hex.EncodeToString(secretBytes)
	digest := sha256.Sum256([]byte(secret))
	credentialID := uuid.NewString()
	credential := meshresource.NewMCPCredentialResourceWithAttributes(credentialID, mesh)
	credential.Spec = &meshproto.MCPCredential{
		ServerId:    serverID,
		DisplayName: displayName,
		SecretHash:  "sha256:" + hex.EncodeToString(digest[:]),
		Status:      "active",
		ExpiresAt:   expiresAt.UTC().Format(time.RFC3339),
		CreatedAt:   time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := r.resources.Add(credential); err != nil {
		return nil, "", err
	}
	return credential, "mcp_" + credentialID + "." + secret, nil
}

func (r *Credentials) List(mesh, serverID string) ([]*meshresource.MCPCredentialResource, error) {
	return manager.ListByIndexes[*meshresource.MCPCredentialResource](r.resources, meshresource.MCPCredentialKind, []index.IndexCondition{
		{IndexName: index.ByMeshIndex, Value: mesh, Operator: index.Equals},
		{IndexName: index.ByMCPCredentialServerID, Value: serverID, Operator: index.Equals},
	})
}

func (r *Credentials) Revoke(mesh, serverID, credentialID, expectedVersion string) (*meshresource.MCPCredentialResource, error) {
	credential, exists, err := manager.GetByKey[*meshresource.MCPCredentialResource](
		r.resources, meshresource.MCPCredentialKind, coremodel.BuildResourceKey(mesh, credentialID))
	if err != nil {
		return nil, err
	}
	if !exists || credential.Spec == nil || credential.Spec.ServerId != serverID {
		return nil, fmt.Errorf("MCP credential %q not found", credentialID)
	}
	updated := credential.DeepCopyObject().(*meshresource.MCPCredentialResource)
	updated.Spec.Status = "revoked"
	if err := r.conditional.CompareAndSwap(updated, expectedVersion); err != nil {
		return nil, err
	}
	return updated, nil
}

func VerifySecretHash(stored, secret string) bool {
	if !strings.HasPrefix(stored, "sha256:") {
		return false
	}
	expected, err := hex.DecodeString(strings.TrimPrefix(stored, "sha256:"))
	if err != nil || len(expected) != sha256.Size {
		return false
	}
	actual := sha256.Sum256([]byte(secret))
	return subtle.ConstantTimeCompare(expected, actual[:]) == 1
}
