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

package discovery

import (
	"sync"
	"sync/atomic"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/core/controller"
	"github.com/apache/dubbo-admin/pkg/core/events"
)

type SyncState struct {
	InitialSyncComplete bool
	Healthy             bool
	Generation          uint64
}

type watchedInformer struct {
	informer   controller.Informer
	health     atomic.Bool
	generation atomic.Uint64
}

type trackedListerWatcher struct {
	controller.ResourceListerWatcher
	health *atomic.Bool
}

func (w *trackedListerWatcher) Close() {
	if closer, ok := w.ResourceListerWatcher.(interface{ Close() }); ok {
		closer.Close()
	}
}

func (w *trackedListerWatcher) List(options metav1.ListOptions) (runtime.Object, error) {
	list, err := w.ResourceListerWatcher.List(options)
	w.health.Store(err == nil)
	return list, err
}

func (w *trackedListerWatcher) Watch(options metav1.ListOptions) (watch.Interface, error) {
	stream, err := w.ResourceListerWatcher.Watch(options)
	if err != nil {
		w.health.Store(false)
	}
	return stream, err
}

func (d *discoveryComponent) SyncState(mesh string) SyncState {
	informers := d.watched[mesh]
	if len(informers) == 0 {
		return SyncState{}
	}
	state := SyncState{InitialSyncComplete: true, Healthy: true}
	for _, item := range informers {
		if !item.informer.HasSynced() || item.informer.IsStopped() {
			state.InitialSyncComplete = false
		}
		if !item.health.Load() || item.informer.IsStopped() {
			state.Healthy = false
		}
		state.Generation += item.generation.Load()
	}
	for _, normalization := range d.normalizations {
		if !normalization.healthy(mesh) {
			state.Healthy = false
		}
	}
	return state
}

type trackedEmitter struct {
	events.Emitter
	generation *atomic.Uint64
}

func (e trackedEmitter) Send(event events.Event) {
	e.generation.Add(1)
	defer e.generation.Add(1)
	e.Emitter.Send(event)
}

// Normalization must finish before an informer can report synchronized. Errors
// remain unhealthy until the failed Resource is successfully reconciled.
type trackedNormalization struct {
	events.Subscriber
	mu       sync.Mutex
	failures map[string]string
}

func (s *trackedNormalization) AsyncEnabled() bool { return false }

func (s *trackedNormalization) ProcessEvent(event events.Event) error {
	resource := event.NewObj()
	if resource == nil {
		resource = event.OldObj()
	}
	err := s.Subscriber.ProcessEvent(event)
	if resource != nil {
		s.mu.Lock()
		if err == nil {
			delete(s.failures, resource.ResourceKey())
		} else {
			s.failures[resource.ResourceKey()] = resource.ResourceMesh()
		}
		s.mu.Unlock()
	}
	return err
}

func (s *trackedNormalization) healthy(mesh string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, failedMesh := range s.failures {
		if failedMesh == mesh {
			return false
		}
	}
	return true
}

func (d *discoveryComponent) trackNormalization(sub events.Subscriber) events.Subscriber {
	tracked := &trackedNormalization{Subscriber: sub, failures: make(map[string]string)}
	d.normalizations = append(d.normalizations, tracked)
	return tracked
}

func (d *discoveryComponent) markWatchError(item *watchedInformer) cache.WatchErrorHandler {
	return func(_ *cache.Reflector, _ error) {
		item.health.Store(false)
	}
}
