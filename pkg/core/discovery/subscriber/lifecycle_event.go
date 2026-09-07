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

package subscriber

import (
	"fmt"
	"time"

	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/common/constants"
	enginecfg "github.com/apache/dubbo-admin/pkg/config/engine"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

// LifecycleEventSubscriber processes LifecycleEvent resources on the EventBus.
// It enriches K8s-sourced events with Dubbo application identification
// and discards events from non-Dubbo Pods before they reach the store.
type LifecycleEventSubscriber struct {
	storeRouter store.Router
	engineCfg   *enginecfg.Config
}

func NewLifecycleEventSubscriber(storeRouter store.Router, engineCfg *enginecfg.Config) *LifecycleEventSubscriber {
	return &LifecycleEventSubscriber{
		storeRouter: storeRouter,
		engineCfg:   engineCfg,
	}
}

func (k *LifecycleEventSubscriber) Name() string {
	return "Discovery-" + k.ResourceKind().ToString()
}

func (k *LifecycleEventSubscriber) ResourceKind() coremodel.ResourceKind {
	return meshresource.LifecycleEventKind
}

func (k *LifecycleEventSubscriber) AsyncEnabled() bool {
	return true
}

func (k *LifecycleEventSubscriber) ProcessEvent(event events.Event) error {
	switch event.Type() {
	case cache.Deleted:
		return k.processDelete(event)
	default:
		return k.processUpsert(event)
	}
}

func (k *LifecycleEventSubscriber) processUpsert(event events.Event) error {
	newObj, ok := event.NewObj().(*meshresource.LifecycleEventResource)
	if !ok || newObj == nil || newObj.Spec == nil {
		logger.Debugf("LifecycleEventSubscriber: event has no valid LifecycleEventResource, skipping")
		return nil
	}

	// REGISTRY events: already enriched by registry subscriber, write directly.
	if newObj.Spec.EventSource == "REGISTRY" {
		return k.writeEvent(newObj)
	}

	// KUBERNETES events: filter by Dubbo app identification.
	identifier := k.engineCfg.Properties.DubboAppIdentifier
	if identifier == nil {
		logger.Debugf("LifecycleEventSubscriber: DubboAppIdentifier not configured, skipping K8s event %s",
			newObj.Spec.InvolvedObjName)
		return nil
	}

	if newObj.Spec.InvolvedObjKind != "Pod" {
		logger.Debugf("LifecycleEventSubscriber: skipping non-Pod K8s event (kind=%s)", newObj.Spec.InvolvedObjKind)
		return nil
	}

	// Align LifecycleEvent mesh with the RuntimeInstance mesh.
	// K8sEventListerWatcher cannot know the Pod's Dubbo mesh at transform time,
	// so we resolve it here by looking up the corresponding RuntimeInstance.
	k.alignMeshFromRuntimeInstance(newObj)

	return k.writeEvent(newObj)
}

// alignMeshFromRuntimeInstance resolves the correct Dubbo mesh for a LifecycleEvent
// by looking up the corresponding RuntimeInstance via the Pod name.
// RuntimeInstances are stored under their discovery mesh (e.g. "nacos2.5"),
// not the engine mesh, so we use the name index to search across all meshes.
func (k *LifecycleEventSubscriber) alignMeshFromRuntimeInstance(eventRes *meshresource.LifecycleEventResource) {
	rtStore, err := k.storeRouter.ResourceKindRoute(meshresource.RuntimeInstanceKind)
	if err != nil {
		logger.Debugf("LifecycleEventSubscriber: cannot route to RuntimeInstance store: %v", err)
		return
	}

	podName := eventRes.Spec.InvolvedObjName

	rtResources, listErr := rtStore.ListByIndexes([]index.IndexCondition{
		{IndexName: index.ByRuntimeInstanceNameIndex, Value: podName, Operator: index.Equals},
	})
	if listErr != nil {
		logger.Debugf("LifecycleEventSubscriber: failed to list RuntimeInstances by name %s: %v", podName, listErr)
		return
	}

	for _, rtRes := range rtResources {
		rtInstance, ok := rtRes.(*meshresource.RuntimeInstanceResource)
		if !ok || rtInstance == nil {
			continue
		}
		resolvedMesh := rtInstance.ResourceMesh()
		if resolvedMesh != "" && resolvedMesh != eventRes.Mesh {
			// Delete the old entry (keyed by old mesh) before changing mesh.
			eventStore, _ := k.storeRouter.ResourceKindRoute(meshresource.LifecycleEventKind)
			if eventStore != nil {
				_ = eventStore.Delete(eventRes)
			}
			eventRes.Mesh = resolvedMesh
			logger.Debugf("LifecycleEventSubscriber: aligned mesh from %q to %q for pod %s",
				k.engineCfg.ID, resolvedMesh, podName)
			return
		}
	}
}

func (k *LifecycleEventSubscriber) processDelete(event events.Event) error {
	oldObj, ok := event.OldObj().(*meshresource.LifecycleEventResource)
	if !ok || oldObj == nil {
		return nil
	}

	eventStore, err := k.storeRouter.ResourceKindRoute(meshresource.LifecycleEventKind)
	if err != nil {
		logger.Errorf("LifecycleEventSubscriber: cannot route to LifecycleEvent store: %v", err)
		return err
	}

	if err := eventStore.Delete(oldObj); err != nil {
		logger.Errorf("LifecycleEventSubscriber: failed to delete LifecycleEvent %s: %v", oldObj.ResourceKey(), err)
	}
	return nil
}

func (k *LifecycleEventSubscriber) writeEvent(eventRes *meshresource.LifecycleEventResource) error {
	eventStore, err := k.storeRouter.ResourceKindRoute(meshresource.LifecycleEventKind)
	if err != nil {
		logger.Errorf("LifecycleEventSubscriber: cannot route to LifecycleEvent store: %v", err)
		return err
	}

	// Ensure K8s-sourced events have a sortable timestamp prefix in their key
	// so that PageListByIndexes (which sorts keys alphabetically) returns
	// events in chronological order.
	var oldName string
	if eventRes.Spec.EventSource == "KUBERNETES" {
		oldName = eventRes.Name
		k.prefixTimestampKey(eventRes)
	}

	if err := eventStore.Add(eventRes); err != nil {
		logger.Errorf("LifecycleEventSubscriber: failed to add LifecycleEvent %s: %v", eventRes.ResourceKey(), err)
		return err
	}

	// Delete the informer-written entry with the original key only after the
	// timestamp-prefixed entry has been successfully added, so that a failed
	// Add does not cause permanent event loss.
	if oldName != "" {
		oldRes := meshresource.NewLifecycleEventResourceWithAttributes(oldName, eventRes.Mesh)
		_ = eventStore.Delete(oldRes)
	}

	logger.Infof("LifecycleEventSubscriber: processed event, source=%s, kind=%s, involved=%s",
		eventRes.Spec.EventSource, eventRes.Spec.InvolvedObjKind, eventRes.Spec.InvolvedObjName)
	return nil
}

// prefixTimestampKey prepends a descending nano-timestamp to the resource name
// so that the store's alphabetical key sort produces chronological (most-recent-first)
// ordering. Registry events already have this prefix from RecordRegistryEvent.
// The caller (writeEvent) is responsible for deleting the original key after a
// successful Add to avoid duplicate entries.
func (k *LifecycleEventSubscriber) prefixTimestampKey(eventRes *meshresource.LifecycleEventResource) {
	timestampNano := int64(0)
	if eventRes.Spec.LastTimestamp != "" {
		if t, err := time.Parse(constants.TimeFormatStr, eventRes.Spec.LastTimestamp); err == nil {
			timestampNano = t.UnixNano()
		}
	}
	if timestampNano == 0 {
		timestampNano = time.Now().UnixNano()
	}
	// Use descending order: more recent → smaller prefix → sorts first alphabetically.
	// maxInt64 - timestampNano gives descending sort.
	sortablePrefix := fmt.Sprintf("%019d", int64(^uint64(0)>>1)-timestampNano)
	eventRes.Name = sortablePrefix + "-" + eventRes.Name
}
