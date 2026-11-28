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

type SyncState struct {
	InitialSyncComplete bool
	Healthy             bool
}

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
	Services  []*meshresource.ServiceResource
	Providers []*meshresource.ServiceProviderMetadataResource
	Instances []*meshresource.RPCInstanceResource
	Sync      SyncState
	Now       time.Time
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

	next := make(map[string]Result)
	i.mu.Lock()
	previousItems := make(map[string]Result, len(i.items))
	for key, value := range i.items {
		previousItems[key] = value
	}
	i.mu.Unlock()
	for identity := range identities {
		for _, protocol := range protocolsForIdentity(identity, snapshot.Providers, snapshot.Instances) {
			result := evaluate(identity, protocol, snapshot)
			key := resultKey(identity, protocol)
			if result.State == Inactive {
				if previous, ok := previousItems[key]; ok && !previous.InactiveSince.IsZero() {
					result.InactiveSince = previous.InactiveSince
				} else {
					result.InactiveSince = now
				}
			}
			next[key] = result
		}
	}

	i.mu.Lock()
	i.items = next
	i.mu.Unlock()

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

	for _, provider := range snapshot.Providers {
		if provider == nil || provider.Spec == nil ||
			provider.Mesh != identity.Mesh ||
			provider.Spec.ServiceName != identity.ServiceName ||
			provider.Spec.Group != identity.Group ||
			provider.Spec.Version != identity.Version {
			continue
		}
		for _, instance := range snapshot.Instances {
			if instance == nil || instance.Spec == nil ||
				instance.Mesh != identity.Mesh ||
				instance.Spec.AppName != provider.Spec.ProviderAppName ||
				instance.Spec.UnregisterTime != "" ||
				!supportsProtocol(instance.Spec.Protocol, instance.Spec.Endpoints, protocol) {
				continue
			}
			result.State = Active
			return result
		}
	}
	result.State = Inactive
	return result
}

func protocolEvidence(identity Identity, snapshot Snapshot) Evidence {
	result := Evidence{}
	providerApps := make(map[string]struct{})
	protocols := make(map[string]struct{})
	complete := snapshot.Sync.InitialSyncComplete && snapshot.Sync.Healthy
	available := false
	for _, provider := range snapshot.Providers {
		if provider == nil || provider.Spec == nil ||
			provider.Mesh != identity.Mesh ||
			provider.Spec.ServiceName != identity.ServiceName ||
			provider.Spec.Group != identity.Group ||
			provider.Spec.Version != identity.Version {
			continue
		}
		providerApps[provider.Spec.ProviderAppName] = struct{}{}
	}
	if len(providerApps) == 0 {
		return result
	}
	seenApps := make(map[string]struct{})
	for _, instance := range snapshot.Instances {
		if instance == nil || instance.Spec == nil || instance.Mesh != identity.Mesh {
			continue
		}
		if _, ok := providerApps[instance.Spec.AppName]; !ok {
			continue
		}
		if instance.Spec.UnregisterTime != "" {
			continue
		}
		available = true
		seenApps[instance.Spec.AppName] = struct{}{}
		for value := range instanceProtocols(instance.Spec.Protocol, instance.Spec.Endpoints) {
			protocols[value] = struct{}{}
		}
	}
	for app := range providerApps {
		if _, ok := seenApps[app]; !ok {
			complete = false
		}
	}
	result.Complete = complete
	result.Available = available
	for protocol := range protocols {
		result.Protocols = append(result.Protocols, protocol)
	}
	sort.Strings(result.Protocols)
	return result
}

func protocolsForIdentity(identity Identity, providers []*meshresource.ServiceProviderMetadataResource, instances []*meshresource.RPCInstanceResource) []string {
	set := map[string]struct{}{"dubbo": {}, "tri": {}}
	for _, provider := range providers {
		if provider == nil || provider.Spec == nil || provider.Mesh != identity.Mesh ||
			provider.Spec.ServiceName != identity.ServiceName || provider.Spec.Group != identity.Group ||
			provider.Spec.Version != identity.Version {
			continue
		}
		for _, instance := range instances {
			if instance != nil && instance.Spec != nil && instance.Mesh == identity.Mesh &&
				instance.Spec.AppName == provider.Spec.ProviderAppName {
				for protocol := range instanceProtocols(instance.Spec.Protocol, instance.Spec.Endpoints) {
					set[protocol] = struct{}{}
				}
			}
		}
	}
	result := make([]string, 0, len(set))
	for protocol := range set {
		result = append(result, protocol)
	}
	sort.Strings(result)
	return result
}

func supportsProtocol(primary string, endpoints []*meshproto.Endpoint, selected string) bool {
	_, ok := instanceProtocols(primary, endpoints)[selected]
	return ok
}

func instanceProtocols(primary string, endpoints []*meshproto.Endpoint) map[string]struct{} {
	result := make(map[string]struct{})
	if value := strings.TrimSpace(primary); value != "" {
		result[value] = struct{}{}
	}
	for _, endpoint := range endpoints {
		if endpoint != nil && strings.TrimSpace(endpoint.Protocol) != "" {
			result[endpoint.Protocol] = struct{}{}
		}
	}
	return result
}

func resultKey(identity Identity, protocol string) string {
	return identity.Mesh + "\x00" + identity.ServiceName + "\x00" +
		identity.Group + "\x00" + identity.Version + "\x00" + protocol
}
