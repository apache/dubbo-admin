/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package availability

import (
	"sort"
	"strings"
	"sync"
	"time"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/discovery"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

type State string

const (
	Active   State = "ACTIVE"
	Inactive State = "INACTIVE"
	Unknown  State = "UNKNOWN"
)

type Identity struct {
	Mesh        string
	ServiceName string
	Group       string
	Version     string
}

type SyncState = discovery.SyncState

type Evidence struct {
	Complete  bool
	Available bool
	Protocols []string
}

type Result struct {
	Identity      Identity
	Protocol      string
	State         State
	InactiveSince time.Time
	Evidence      Evidence
}

type Snapshot struct {
	// Identities includes published bindings even after their discovery Resources disappear.
	Identities []Identity
	Services   []*meshresource.ServiceResource
	Providers  []*meshresource.ServiceProviderMetadataResource
	Instances  []*meshresource.RPCInstanceResource
	Metadata   []*meshresource.RPCInstanceMetadataResource
	Sync       SyncState
	Now        time.Time
}

type LiveServiceIndex struct {
	grace time.Duration
	mu    sync.Mutex
	items map[string]Result
}

func NewLiveServiceIndex(grace time.Duration) *LiveServiceIndex {
	if grace <= 0 {
		grace = 60 * time.Second
	}
	return &LiveServiceIndex{grace: grace, items: make(map[string]Result)}
}

func (i *LiveServiceIndex) Refresh(snapshot Snapshot) []Result {
	now := snapshot.Now
	if now.IsZero() {
		now = time.Now()
	}
	identities := make(map[Identity]struct{})
	for _, identity := range snapshot.Identities {
		identities[identity] = struct{}{}
	}
	for _, service := range snapshot.Services {
		if service != nil && service.Spec != nil {
			identities[Identity{
				Mesh: service.Mesh, ServiceName: service.Spec.Name,
				Group: service.Spec.Group, Version: service.Spec.Version,
			}] = struct{}{}
		}
	}
	for _, provider := range snapshot.Providers {
		if provider != nil && provider.Spec != nil {
			identities[Identity{
				Mesh: provider.Mesh, ServiceName: provider.Spec.ServiceName,
				Group: provider.Spec.Group, Version: provider.Spec.Version,
			}] = struct{}{}
		}
	}

	i.mu.Lock()
	defer i.mu.Unlock()
	for _, previous := range i.items {
		identities[previous.Identity] = struct{}{}
	}
	next := make(map[string]Result)
	for identity := range identities {
		for _, protocol := range []string{"dubbo", "tri"} {
			result := evaluate(identity, protocol, snapshot)
			key := resultKey(identity, protocol)
			if result.State == Inactive {
				if previous, ok := i.items[key]; ok && previous.State == Inactive {
					result.InactiveSince = previous.InactiveSince
				} else {
					result.InactiveSince = now
				}
			}
			next[key] = result
		}
	}

	i.items = next

	results := make([]Result, 0, len(next))
	for _, result := range next {
		results = append(results, result)
	}
	sort.Slice(results, func(a, b int) bool {
		return resultKey(results[a].Identity, results[a].Protocol) <
			resultKey(results[b].Identity, results[b].Protocol)
	})
	return results
}

// FilterTools applies availability to a current published tool list. Call it on
// every tools/list, including when the compiled handler is cached.
func (i *LiveServiceIndex) FilterTools(mesh string, tools []*meshproto.MCPToolBinding, snapshot Snapshot) []*meshproto.MCPToolBinding {
	for _, tool := range tools {
		if tool != nil && tool.Operation != nil {
			snapshot.Identities = append(snapshot.Identities, Identity{
				Mesh: mesh, ServiceName: tool.Operation.ServiceName,
				Group: tool.Operation.Group, Version: tool.Operation.Version,
			})
		}
	}
	results := make(map[string]Result)
	for _, result := range i.Refresh(snapshot) {
		results[resultKey(result.Identity, result.Protocol)] = result
	}
	visible := make([]*meshproto.MCPToolBinding, 0, len(tools))
	for _, tool := range tools {
		if tool == nil || tool.Operation == nil {
			continue
		}
		identity := Identity{
			Mesh: mesh, ServiceName: tool.Operation.ServiceName,
			Group: tool.Operation.Group, Version: tool.Operation.Version,
		}
		if i.Visible(results[resultKey(identity, tool.Operation.Protocol)], snapshot.Now) {
			visible = append(visible, tool)
		}
	}
	return visible
}

func (i *LiveServiceIndex) Get(identity Identity, protocol string) (Result, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	result, ok := i.items[resultKey(identity, protocol)]
	return result, ok
}

func (i *LiveServiceIndex) Visible(result Result, now time.Time) bool {
	if result.State != Inactive || result.InactiveSince.IsZero() {
		return true
	}
	if now.IsZero() {
		now = time.Now()
	}
	return now.Sub(result.InactiveSince) <= i.grace
}

func Evaluate(snapshot Snapshot, identity Identity, protocol string) Result {
	return evaluate(identity, protocol, snapshot)
}

func ConfirmedProtocolIncompatibility(result Result) bool {
	if !result.Evidence.Complete || !result.Evidence.Available {
		return false
	}
	for _, protocol := range result.Evidence.Protocols {
		if protocol == result.Protocol {
			return false
		}
	}
	return true
}

func evaluate(identity Identity, protocol string, snapshot Snapshot) Result {
	result := Result{
		Identity: identity,
		Protocol: protocol,
		State:    Unknown,
	}
	result.Evidence = protocolEvidence(identity, snapshot)
	if !snapshot.Sync.InitialSyncComplete || !snapshot.Sync.Healthy {
		return result
	}
	if result.Evidence.Available {
		for _, availableProtocol := range result.Evidence.Protocols {
			if availableProtocol == protocol {
				result.State = Active
				return result
			}
		}
	}
	if result.Evidence.Complete {
		result.State = Inactive
	}
	return result
}

func protocolEvidence(identity Identity, snapshot Snapshot) Evidence {
	protocols := make(map[string]struct{})
	complete := snapshot.Sync.InitialSyncComplete && snapshot.Sync.Healthy
	result := Evidence{}
	for _, instance := range snapshot.Instances {
		if instance == nil || instance.Spec == nil || instance.Mesh != identity.Mesh {
			continue
		}
		if instance.Spec.UnregisterTime != "" || (instance.Spec.Available != nil && !*instance.Spec.Available) {
			continue
		}
		exports, known := exportsForInstance(instance, snapshot.Metadata)
		if !known {
			complete = false
			continue
		}
		for _, export := range exports {
			if export == nil || export.Name != identity.ServiceName ||
				export.Group != identity.Group || export.Version != identity.Version {
				continue
			}
			result.Available = true
			if value := strings.TrimSpace(export.Protocol); value != "" {
				protocols[value] = struct{}{}
			} else {
				complete = false
			}
		}
	}
	for _, registration := range snapshot.Metadata {
		if registration == nil || registration.Spec == nil ||
			registration.Mesh != identity.Mesh || registration.Spec.InstanceName == "" {
			continue
		}
		if registration.Spec.Available == nil {
			complete = false
			continue
		}
		if !*registration.Spec.Available {
			continue
		}
		for _, export := range registration.Spec.Services {
			if export == nil || export.Name != identity.ServiceName ||
				export.Group != identity.Group || export.Version != identity.Version {
				continue
			}
			result.Available = true
			if value := strings.TrimSpace(export.Protocol); value != "" {
				protocols[value] = struct{}{}
			} else {
				complete = false
			}
		}
	}
	result.Complete = complete
	for protocol := range protocols {
		result.Protocols = append(result.Protocols, protocol)
	}
	sort.Strings(result.Protocols)
	return result
}

func exportsForInstance(
	instance *meshresource.RPCInstanceResource,
	metadata []*meshresource.RPCInstanceMetadataResource,
) ([]*meshproto.ServiceInfo, bool) {
	if instance.Spec.Revision == "" {
		return nil, false
	}
	for _, item := range metadata {
		if item == nil || item.Spec == nil || item.Mesh != instance.Mesh ||
			item.Spec.App != instance.Spec.AppName || item.Spec.Revision != instance.Spec.Revision {
			continue
		}
		exports := make([]*meshproto.ServiceInfo, 0, len(item.Spec.Services))
		for _, service := range item.Spec.Services {
			exports = append(exports, service)
		}
		return exports, true
	}
	return nil, false
}

func resultKey(identity Identity, protocol string) string {
	return identity.Mesh + "\x00" + identity.ServiceName + "\x00" +
		identity.Group + "\x00" + identity.Version + "\x00" + protocol
}
