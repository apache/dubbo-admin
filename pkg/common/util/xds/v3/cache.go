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

// Copyright 2018 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package v3

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	envoyconfigcorev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	envoycache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/log"
	"github.com/envoyproxy/go-control-plane/pkg/server/stream/v3"
)

type Snapshot interface {
	// GetSupportedTypes returns a list of xDS types supported by this snapshot.
	GetSupportedTypes() []string

	// Consistent check verifies that the dependent resources are exactly listed in the
	// snapshot:
	// - all EDS resources are listed by name in CDS resources
	// - all RDS resources are listed by name in LDS resources
	//
	// Note that clusters and listeners are requested without name references, so
	// Envoy will accept the snapshot list of clusters as-is even if it does not match
	// all references found in xDS.
	Consistent() error

	// GetResources selects snapshot resources by type.
	GetResources(typ string) map[string]types.Resource

	// GetVersion returns the version for a resource type.
	GetVersion(typ string) string

	// WithVersion creates a new snapshot with a different version for a given resource type.
	WithVersion(typ string, version string) Snapshot
}

// SnapshotCache is a snapshot-based envoy_cache that maintains a single versioned
// snapshot of responses per node. SnapshotCache consistently replies with the
// latest snapshot. For the protocol to work correctly in ADS mode, EDS/RDS
// requests are responded only when all resources in the snapshot xDS response
// are named as part of the request. It is expected that the CDS response names
// all EDS clusters, and the LDS response names all RDS routes in a snapshot,
// to ensure that Envoy makes the request for all EDS clusters or RDS routes
// eventually.
//
// SnapshotCache can operate as a REST or regular xDS backend. The snapshot
// can be partial, e.g. only include RDS or EDS resources.
type SnapshotCache interface {
	envoycache.Cache

	// SetSnapshot sets a response snapshot for a node. For ADS, the snapshots
	// should have distinct versions and be internally consistent (e.g. all
	// referenced resources must be included in the snapshot).
	//
	// This method will cause the server to respond to all open watches, for which
	// the version differs from the snapshot version.
	SetSnapshot(node string, snapshot Snapshot) error

	// GetSnapshot gets the snapshot for a node.
	GetSnapshot(node string) (Snapshot, error)

	// HasSnapshot checks whether there is a snapshot present for a node.
	HasSnapshot(node string) bool

	// ClearSnapshot removes all status and snapshot information associated with a node. Return the removed snapshot or nil
	ClearSnapshot(node string) Snapshot

	// GetStatusInfo retrieves status information for a node ID.
	GetStatusInfo(string) StatusInfo

	// GetStatusKeys retrieves node IDs for all statuses.
	GetStatusKeys() []string
}

// Generates a snapshot of xDS resources for a given node.
type SnapshotGenerator interface {
	GenerateSnapshot(context.Context, *envoyconfigcorev3.Node) (Snapshot, error)
}

type snapshotCache struct {
	// watchCount is an atomic counter incremented for each watch. This needs to
	// be the first field in the struct to guarantee that it is 64-bit aligned,
	// which is a requirement for atomic operations on 64-bit operands to work on
	// 32-bit machines.
	watchCount int64

	log log.Logger

	// ads flag to hold responses until all resources are named
	ads bool

	// snapshots are cached resources indexed by node IDs
	snapshots map[string]Snapshot

	// status information for all nodes indexed by node IDs
	status map[string]*statusInfo

	// hash is the hashing function for Envoy nodes
	hash NodeHash

	mu sync.RWMutex
}

// NewSnapshotCache initializes a simple envoy_cache.
//
// ADS flag forces a delay in responding to streaming requests until all
// resources are explicitly named in the request. This avoids the problem of a
// partial request over a single stream for a subset of resources which would
// require generating a fresh version for acknowledgement. ADS flag requires
// snapshot consistency. For non-ADS case (and fetch), multiple partial
// requests are sent across multiple streams and re-using the snapshot version
// is OK.
//
// Logger is optional.
func NewSnapshotCache(ads bool, hash NodeHash, logger log.Logger) SnapshotCache {
	return &snapshotCache{
		log:       logger,
		ads:       ads,
		snapshots: make(map[string]Snapshot),
		status:    make(map[string]*statusInfo),
		hash:      hash,
	}
}

// SetSnapshotCache updates a snapshot for a node.
func (cache *snapshotCache) SetSnapshot(node string, snapshot Snapshot) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	// update the existing entry
	cache.snapshots[node] = snapshot

	// trigger existing watches for which version changed
	if info, ok := cache.status[node]; ok {
		info.mu.Lock()
		for id, watch := range info.watches {
			version := snapshot.GetVersion(watch.Request.TypeUrl)
			if version != watch.Request.VersionInfo {
				if cache.log != nil {
					cache.log.Debugf("respond open watch %d%v with new version %q", id, watch.Request.ResourceNames, version)
				}
				cache.respond(watch.Request, watch.Response, snapshot.GetResources(watch.Request.TypeUrl), version)

				// discard the watch
				delete(info.watches, id)
			}
		}
		info.mu.Unlock()
	}

	return nil
}

// GetSnapshots gets the snapshot for a node, and returns an error if not found.
func (cache *snapshotCache) GetSnapshot(node string) (Snapshot, error) {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	snap, ok := cache.snapshots[node]
	if !ok {
		return nil, fmt.Errorf("no snapshot found for node %s", node)
	}
	return snap, nil
}

func (cache *snapshotCache) HasSnapshot(node string) bool {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	_, ok := cache.snapshots[node]
	return ok
}

// ClearSnapshot clears snapshot and info for a node.
func (cache *snapshotCache) ClearSnapshot(node string) Snapshot {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	snapshot := cache.snapshots[node]
	delete(cache.snapshots, node)
	delete(cache.status, node)
	return snapshot
}

// nameSet creates a map from a string slice to value true.
func nameSet(names []string) map[string]bool {
	set := make(map[string]bool)
	for _, name := range names {
		set[name] = true
	}
	return set
}

// superset checks that all resources are listed in the names set.
func superset(names map[string]bool, resources map[string]types.Resource) error {
	for resourceName := range resources {
		if _, exists := names[resourceName]; !exists {
			return fmt.Errorf("%q not listed", resourceName)
		}
	}
	return nil
}

func (cache *snapshotCache) CreateDeltaWatch(*envoycache.DeltaRequest, stream.StreamState, chan envoycache.DeltaResponse) func() {
	return nil
}

// CreateWatch returns a watch for an xDS request.
func (cache *snapshotCache) CreateWatch(request *envoycache.Request, _ stream.StreamState, responseChan chan envoycache.Response) func() {
	nodeID := cache.hash.ID(request.Node)

	cache.mu.Lock()
	defer cache.mu.Unlock()
	info, ok := cache.status[nodeID]
	if !ok {
		info = newStatusInfo(request.Node)
		cache.status[nodeID] = info
	}

	// update last watch request time
	info.mu.Lock()
	info.lastWatchRequestTime = time.Now()
	info.mu.Unlock()

	snapshot, exists := cache.snapshots[nodeID]
	version := ""
	if exists {
		version = snapshot.GetVersion(request.TypeUrl)
	}

	// if the requested version is up-to-date or missing a response, leave an open watch
	if !exists || request.VersionInfo == version {
		watchID := cache.nextWatchID()
		if cache.log != nil {
			cache.log.Debugf("open watch %d for %s%v from nodeID %q, version %q", watchID,
				request.TypeUrl, request.ResourceNames, nodeID, request.VersionInfo)
		}
		info.mu.Lock()
		info.watches[watchID] = ResponseWatch{Request: request, Response: responseChan}
		info.mu.Unlock()
		return cache.cancelWatch(nodeID, watchID)
	}

	// otherwise, the watch may be responded immediately
	cache.respond(request, responseChan, snapshot.GetResources(request.TypeUrl), version)

	return nil
}

func (cache *snapshotCache) nextWatchID() int64 {
	return atomic.AddInt64(&cache.watchCount, 1)
}

// cancellation function for cleaning stale watches
func (cache *snapshotCache) cancelWatch(nodeID string, watchID int64) func() {
	return func() {
		// uses the envoy_cache mutex
		cache.mu.Lock()
		defer cache.mu.Unlock()
		if info, ok := cache.status[nodeID]; ok {
			info.mu.Lock()
			delete(info.watches, watchID)
			info.mu.Unlock()
		}
	}
}

// Respond to a watch with the snapshot value. The value channel should have capacity not to block.
// TODO(kuat) do not respond always, see issue https://github.com/envoyproxy/go-control-plane/issues/46
func (cache *snapshotCache) respond(request *envoycache.Request, value chan envoycache.Response, resources map[string]types.Resource, version string) {
	// for ADS, the request names must match the snapshot names
	// if they do not, then the watch is never responded, and it is expected that envoy makes another request
	if len(request.ResourceNames) != 0 && cache.ads {
		if err := superset(nameSet(request.ResourceNames), resources); err != nil {
			if cache.log != nil {
				cache.log.Debugf("ADS mode: not responding to request: %v", err)
			}
			return
		}
	}
	if cache.log != nil {
		cache.log.Debugf("respond %s%v version %q with version %q",
			request.TypeUrl, request.ResourceNames, request.VersionInfo, version)
	}

	value <- createResponse(request, resources, version)
}

func createResponse(request *envoycache.Request, resources map[string]types.Resource, version string) envoycache.Response {
	filtered := make([]types.ResourceWithTTL, 0, len(resources))

	// Reply only with the requested resources. Envoy may ask each resource
	// individually in a separate stream. It is ok to reply with the same version
	// on separate streams since requests do not share their response versions.
	if len(request.ResourceNames) != 0 {
		set := nameSet(request.ResourceNames)
		for name, resource := range resources {
			if set[name] {
				filtered = append(filtered, types.ResourceWithTTL{Resource: resource})
			}
		}
	} else {
		for _, resource := range resources {
			filtered = append(filtered, types.ResourceWithTTL{Resource: resource})
		}
	}

	return &envoycache.RawResponse{
		Request:   request,
		Version:   version,
		Resources: filtered,
	}
}

// Fetch implements the envoy_cache fetch function.
// Fetch is called on multiple streams, so responding to individual names with the same version works.
// If there is a Deadline set on the context, the call will block until either the context is terminated
// or there is a new update.
func (cache *snapshotCache) Fetch(ctx context.Context, request *envoycache.Request) (envoycache.Response, error) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return cache.blockingFetch(ctx, request)
	}

	nodeID := cache.hash.ID(request.Node)

	cache.mu.RLock()
	defer cache.mu.RUnlock()

	if snapshot, exists := cache.snapshots[nodeID]; exists {
		// Respond only if the request version is distinct from the current snapshot state.
		// It might be beneficial to hold the request since Envoy will re-attempt the refresh.
		version := snapshot.GetVersion(request.TypeUrl)
		if request.VersionInfo == version {
			if cache.log != nil {
				cache.log.Warnf("skip fetch: version up to date")
			}
			return nil, &types.SkipFetchError{}
		}

		resources := snapshot.GetResources(request.TypeUrl)
		out := createResponse(request, resources, version)
		return out, nil
	}

	return nil, fmt.Errorf("missing snapshot for %q", nodeID)
}

// blockingFetch will wait until either the context is terminated or new resources become available
func (cache *snapshotCache) blockingFetch(ctx context.Context, request *envoycache.Request) (envoycache.Response, error) {
	responseChan := make(chan envoycache.Response, 1)
	cancelFunc := cache.CreateWatch(request, stream.StreamState{}, responseChan)
	if cancelFunc != nil {
		defer cancelFunc()
	}

	select {
	case <-ctx.Done():
		// finished without an update
		return nil, &types.SkipFetchError{}
	case resp := <-responseChan:
		return resp, nil
	}
}

// GetStatusInfo retrieves the status info for the node.
func (cache *snapshotCache) GetStatusInfo(node string) StatusInfo {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	info, exists := cache.status[node]
	if !exists {
		if cache.log != nil {
			cache.log.Warnf("node does not exist")
		}
		return nil
	}

	return info
}

// GetStatusKeys retrieves all node IDs in the status map.
func (cache *snapshotCache) GetStatusKeys() []string {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	out := make([]string, 0, len(cache.status))
	for id := range cache.status {
		out = append(out, id)
	}

	return out
}

// NodeHash computes string identifiers for Envoy nodes.
type NodeHash interface {
	// ID function defines a unique string identifier for the remote Envoy node.
	ID(node *envoyconfigcorev3.Node) string
}

// IDHash uses ID field as the node hash.
type IDHash struct{}

// ID uses the node ID field
func (IDHash) ID(node *envoyconfigcorev3.Node) string {
	if node == nil {
		return ""
	}
	return node.Id
}

var _ NodeHash = IDHash{}

// StatusInfo tracks the server state for the remote Envoy node.
// Not all fields are used by all envoy_cache implementations.
type StatusInfo interface {
	// GetNode returns the node metadata.
	GetNode() *envoyconfigcorev3.Node

	// GetNumWatches returns the number of open watches.
	GetNumWatches() int

	// GetLastWatchRequestTime returns the timestamp of the last discoveryengine watch request.
	GetLastWatchRequestTime() time.Time
}

type statusInfo struct {
	// node is the constant Envoy node metadata.
	node *envoyconfigcorev3.Node

	// watches are indexed channels for the response watches and the original requests.
	watches map[int64]ResponseWatch

	// the timestamp of the last watch request
	lastWatchRequestTime time.Time

	// mutex to protect the status fields.
	// should not acquire mutex of the parent envoy_cache after acquiring this mutex.
	mu sync.RWMutex
}

// ResponseWatch is a watch record keeping both the request and an open channel for the response.
type ResponseWatch struct {
	// Request is the original request for the watch.
	Request *envoycache.Request

	// Response is the channel to push responses to.
	Response chan envoycache.Response
}

// newStatusInfo initializes a status info data structure.
func newStatusInfo(node *envoyconfigcorev3.Node) *statusInfo {
	out := statusInfo{
		node:    node,
		watches: make(map[int64]ResponseWatch),
	}
	return &out
}

func (info *statusInfo) GetNode() *envoyconfigcorev3.Node {
	info.mu.RLock()
	defer info.mu.RUnlock()
	return info.node
}

func (info *statusInfo) GetNumWatches() int {
	info.mu.RLock()
	defer info.mu.RUnlock()
	return len(info.watches)
}

func (info *statusInfo) GetLastWatchRequestTime() time.Time {
	info.mu.RLock()
	defer info.mu.RUnlock()
	return info.lastWatchRequestTime
}
