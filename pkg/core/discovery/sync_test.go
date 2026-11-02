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
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/core/controller"
	"github.com/apache/dubbo-admin/pkg/core/events"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/store/memory"
)

type normalizationStub struct {
	events.Subscriber
	process func(events.Event) error
}

func (s normalizationStub) ProcessEvent(e events.Event) error { return s.process(e) }

type emitterFunc func(events.Event)

func (f emitterFunc) Send(e events.Event) { f(e) }

func TestSyncStateWaitsForNormalizationAndRecoversAfterReconciliation(t *testing.T) {
	st := memory.NewMemoryResourceStore(meshresource.NacosServiceKind)
	require.NoError(t, st.Init(nil))
	resource := meshresource.NewNacosServiceResourceWithAttributes("orders", "mesh-1")
	stream := watch.NewRaceFreeFake()
	defer stream.Stop()
	list := &cache.ListWatch{
		ListFunc: func(metav1.ListOptions) (k8sruntime.Object, error) {
			return meshresource.NewNacosServiceResourceListWithItems(resource), nil
		},
		WatchFunc: func(metav1.ListOptions) (watch.Interface, error) { return stream, nil },
	}
	entered, release := make(chan struct{}), make(chan struct{})
	var failed atomic.Bool
	failed.Store(true)
	d := newDiscoveryComponent().(*discoveryComponent)
	normalization := d.trackNormalization(normalizationStub{process: func(events.Event) error {
		if failed.Load() {
			close(entered)
			<-release
			return errors.New("metadata unavailable")
		}
		return nil
	}})
	require.False(t, normalization.AsyncEnabled())
	item := &watchedInformer{}
	item.health.Store(true)
	item.informer = controller.NewInformerWithOptions(list,
		trackedEmitter{Emitter: emitterFunc(func(e events.Event) { _ = normalization.ProcessEvent(e) }), generation: &item.generation},
		st, keyFunc, controller.Options{})
	d.watched["mesh-1"] = []*watchedInformer{item}
	require.False(t, d.SyncState("mesh-1").InitialSyncComplete)
	stop := make(chan struct{})
	defer close(stop)
	go item.informer.Run(stop)
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("normalization never started")
	}
	observed := make(chan SyncState, 1)
	go func() { observed <- d.SyncState("mesh-1") }()
	select {
	case <-observed:
		t.Fatal("initial synchronization completed before normalization")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	var state SyncState
	select {
	case state = <-observed:
	case <-time.After(3 * time.Second):
		t.Fatal("synchronization query deadlocked")
	}
	require.True(t, state.InitialSyncComplete)
	require.False(t, state.Healthy)
	require.False(t, d.SyncState("another-mesh").InitialSyncComplete)
	failed.Store(false)
	stream.Modify(resource)
	require.Eventually(t, func() bool { return d.SyncState("mesh-1").Healthy }, 3*time.Second, time.Millisecond)
	require.Greater(t, d.SyncState("mesh-1").Generation, state.Generation)
	item.health.Store(false)
	require.False(t, d.SyncState("mesh-1").Healthy)
	item.health.Store(true)
	require.True(t, d.SyncState("mesh-1").Healthy)
}

func TestNormalizationFailureIsScopedAndNeedsSameResourceToRecover(t *testing.T) {
	fail := true
	s := &trackedNormalization{Subscriber: normalizationStub{process: func(events.Event) error {
		if fail {
			return errors.New("offline")
		}
		return nil
	}}, failures: make(map[string]string)}
	first := meshresource.NewNacosServiceResourceWithAttributes("first", "mesh-1")
	second := meshresource.NewNacosServiceResourceWithAttributes("second", "mesh-1")
	require.Error(t, s.ProcessEvent(events.NewResourceChangedEvent(cache.Added, nil, first)))
	require.False(t, s.healthy("mesh-1"))
	require.True(t, s.healthy("mesh-2"))
	fail = false
	require.NoError(t, s.ProcessEvent(events.NewResourceChangedEvent(cache.Added, nil, second)))
	require.False(t, s.healthy("mesh-1"))
	require.NoError(t, s.ProcessEvent(events.NewResourceChangedEvent(cache.Deleted, first, nil)))
	require.True(t, s.healthy("mesh-1"))
}
