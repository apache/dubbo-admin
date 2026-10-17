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
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

// Servers persists the editable Draft and effective Published snapshot as one
// CAS-protected aggregate. Contract validation is supplied by the caller.
type Servers struct {
	resources   manager.ResourceManager
	conditional manager.ConditionalResourceManager
	cascading   manager.CascadingResourceManager
}

func NewServers(resources manager.ResourceManager) (*Servers, error) {
	conditional, ok := resources.(manager.ConditionalResourceManager)
	if !ok {
		return nil, fmt.Errorf("resource manager does not support conditional mutations")
	}
	cascading, ok := resources.(manager.CascadingResourceManager)
	if !ok {
		return nil, fmt.Errorf("resource manager does not support cascading deletion")
	}
	return &Servers{resources: resources, conditional: conditional, cascading: cascading}, nil
}

func (r *Servers) Create(mesh, name string, draft *meshproto.MCPServerSnapshot) (*meshresource.MCPServerResource, error) {
	if mesh == "" || name == "" || draft == nil {
		return nil, fmt.Errorf("mesh, name and draft are required")
	}
	server := meshresource.NewMCPServerResourceWithAttributes(name, mesh)
	server.Spec.Draft = proto.Clone(draft).(*meshproto.MCPServerSnapshot)
	if err := r.resources.Add(server); err != nil {
		return nil, err
	}
	return server, nil
}

func (r *Servers) Get(mesh, name string) (*meshresource.MCPServerResource, error) {
	server, exists, err := manager.GetByKey[*meshresource.MCPServerResource](
		r.resources, meshresource.MCPServerKind, coremodel.BuildResourceKey(mesh, name))
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("MCP server %q not found", name)
	}
	return server, nil
}

func (r *Servers) UpdateDraft(mesh, name, expectedVersion string, draft *meshproto.MCPServerSnapshot) (*meshresource.MCPServerResource, error) {
	if draft == nil {
		return nil, fmt.Errorf("draft is required")
	}
	server, err := r.Get(mesh, name)
	if err != nil {
		return nil, err
	}
	updated := server.DeepCopyObject().(*meshresource.MCPServerResource)
	updated.Spec.Draft = proto.Clone(draft).(*meshproto.MCPServerSnapshot)
	if err := r.conditional.CompareAndSwap(updated, expectedVersion); err != nil {
		return nil, err
	}
	return updated, nil
}

// Publish validates a copy of the current Draft and replaces the complete
// Published snapshot in one conditional write. The validator may add compiled
// schema fields to the copy; a failure leaves the stored resource untouched.
func (r *Servers) Publish(mesh, name, expectedVersion string, validate func(*meshproto.MCPServerSnapshot) error) (*meshresource.MCPServerResource, error) {
	if validate == nil {
		return nil, fmt.Errorf("publish validator is required")
	}
	server, err := r.Get(mesh, name)
	if err != nil {
		return nil, err
	}
	if server.Spec == nil || server.Spec.Draft == nil {
		return nil, fmt.Errorf("server draft is required")
	}
	snapshot := proto.Clone(server.Spec.Draft).(*meshproto.MCPServerSnapshot)
	if err := validate(snapshot); err != nil {
		return nil, err
	}
	updated := server.DeepCopyObject().(*meshresource.MCPServerResource)
	var revision int64 = 1
	if updated.Spec.Published != nil {
		revision = updated.Spec.Published.Revision + 1
	}
	updated.Spec.Published = &meshproto.MCPPublishedSnapshot{
		Revision:    revision,
		PublishedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Snapshot:    snapshot,
	}
	if err := r.conditional.CompareAndSwap(updated, expectedVersion); err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *Servers) Delete(mesh, name, expectedVersion string) error {
	server := meshresource.NewMCPServerResourceWithAttributes(name, mesh)
	return r.cascading.CompareAndDeleteWithDependents(server, expectedVersion, meshresource.MCPCredentialKind, []index.IndexCondition{
		{IndexName: index.ByMeshIndex, Value: mesh, Operator: index.Equals},
		{IndexName: index.ByMCPCredentialServerID, Value: name, Operator: index.Equals},
	})
}
