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
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	discoverycfg "github.com/apache/dubbo-admin/pkg/config/discovery"
	logcfg "github.com/apache/dubbo-admin/pkg/config/log"
	"github.com/apache/dubbo-admin/pkg/core/controller"
	"github.com/apache/dubbo-admin/pkg/core/discovery/subscriber"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/store/memory"
)

type catalogTestFactory struct {
	lw controller.ResourceListerWatcher
}

func (f *catalogTestFactory) Support(t discoverycfg.Type) bool { return t == "catalog-test" }
func (f *catalogTestFactory) NewListWatchers(_ *discoverycfg.Config) ([]controller.ResourceListerWatcher, error) {
	return []controller.ResourceListerWatcher{f.lw}, nil
}

type catalogTestLW struct {
	listStarted, allowList, watchStarted chan struct{}
	watcher                              *watch.RaceFreeFakeWatcher
}

func (l *catalogTestLW) ResourceKind() coremodel.ResourceKind {
	return meshresource.ServiceProviderMetadataKind
}
func (l *catalogTestLW) TransformFunc() cache.TransformFunc { return nil }
func (l *catalogTestLW) List(_ metav1.ListOptions) (k8sruntime.Object, error) {
	close(l.listStarted)
	<-l.allowList
	return meshresource.NewServiceProviderMetadataResourceListWithItems(), nil
}
func (l *catalogTestLW) Watch(_ metav1.ListOptions) (watch.Interface, error) {
	close(l.watchStarted)
	return l.watcher, nil
}

type relistTestLW struct {
	watchStarted chan int
	watchers     [2]*watch.RaceFreeFakeWatcher
	watchCount   int
}

func (l *relistTestLW) ResourceKind() coremodel.ResourceKind {
	return meshresource.ServiceProviderMetadataKind
}
func (l *relistTestLW) TransformFunc() cache.TransformFunc { return nil }
func (l *relistTestLW) List(_ metav1.ListOptions) (k8sruntime.Object, error) {
	return meshresource.NewServiceProviderMetadataResourceListWithItems(), nil
}
func (l *relistTestLW) Watch(_ metav1.ListOptions) (watch.Interface, error) {
	i := l.watchCount
	l.watchCount++
	if i >= len(l.watchers) {
		return nil, fmt.Errorf("unexpected watch %d", i)
	}
	l.watchStarted <- i
	return l.watchers[i], nil
}

type catalogTestRouter map[coremodel.ResourceKind]store.ResourceStore

func (r catalogTestRouter) ResourceKindRoute(k coremodel.ResourceKind) (store.ResourceStore, error) {
	s, ok := r[k]
	if !ok {
		return nil, fmt.Errorf("no store for %s", k)
	}
	return s, nil
}
func (r catalogTestRouter) ResourceRoute(obj coremodel.Resource) (store.ResourceStore, error) {
	return r.ResourceKindRoute(obj.ResourceKind())
}

type catalogTestBus struct{ ch chan events.Event }

func (b *catalogTestBus) Send(e events.Event)                 { b.ch <- e }
func (b *catalogTestBus) Subscribe(events.Subscriber) error   { return nil }
func (b *catalogTestBus) Unsubscribe(events.Subscriber) error { return nil }

func nextCatalogEvent(t *testing.T, ch <-chan events.Event) events.Event {
	t.Helper()
	select {
	case e := <-ch:
		return e
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for informer event")
		return nil
	}
}

func awaitCatalogSignal(t *testing.T, ch <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(10 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func startCatalogInformer(t *testing.T, mesh string) (store.ResourceStore, catalogTestRouter, *catalogTestBus, *catalogTestLW, func()) {
	t.Helper()
	providerStore := memory.NewMemoryResourceStore(meshresource.ServiceProviderMetadataKind)
	lifecycleStore := memory.NewMemoryResourceStore(meshresource.LifecycleEventKind)
	require.NoError(t, providerStore.Init(nil))
	require.NoError(t, lifecycleStore.Init(nil))
	router := catalogTestRouter{meshresource.ServiceProviderMetadataKind: providerStore, meshresource.LifecycleEventKind: lifecycleStore}
	bus := &catalogTestBus{ch: make(chan events.Event, 20)}
	lw := &catalogTestLW{listStarted: make(chan struct{}), allowList: make(chan struct{}), watchStarted: make(chan struct{}), watcher: watch.NewRaceFreeFake()}
	oldRegistry := registry
	registry = newDiscoveryFactoryRegistry()
	t.Cleanup(func() { registry = oldRegistry })
	RegisterListWatcherFactory(&catalogTestFactory{lw: lw})
	informers, err := (&discoveryComponent{}).initInformers(&discoverycfg.Config{ID: mesh, Type: "catalog-test"}, router, bus)
	require.NoError(t, err)

	var releaseOnce sync.Once
	releaseList := func() { releaseOnce.Do(func() { close(lw.allowList) }) }
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() { defer close(done); informers[0].Run(stop) }()
	t.Cleanup(func() {
		close(stop)
		releaseList()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("timed out waiting for informer to stop")
		}
	})
	return providerStore, router, bus, lw, releaseList
}

func ingestCatalogZKMetadata(t *testing.T, bus *catalogTestBus, router catalogTestRouter) coremodel.Resource {
	t.Helper()
	raw := meshresource.NewZKMetadataResourceWithAttributes("/dubbo/metadata/greet.GreetService/provider/app", "zk-local")
	raw.Spec = &meshproto.ZKMetadata{NodePath: raw.Name, NodeData: `{"canonicalName":"greet.GreetService","parameters":{"application":"app"}}`}
	err := subscriber.NewZKMetadataEventSubscriber(bus, router).ProcessEvent(events.NewResourceChangedEvent(cache.Added, nil, raw))
	require.NoError(t, err)
	e := nextCatalogEvent(t, bus.ch)
	require.Equal(t, cache.Added, e.Type())
	return e.NewObj()
}

func TestDiscoveryInformerCleanupReleasesBlockedList(t *testing.T) {
	logger.Init(&logcfg.Config{Level: logcfg.LevelError, OutputPath: filepath.Join(t.TempDir(), "test.log")})
	_, _, _, lw, _ := startCatalogInformer(t, "mesh")
	awaitCatalogSignal(t, lw.listStarted, "list to start")
	// Return with List still blocked; cleanup must release it and stop the informer.
}

func TestDiscoveryInformerKeepsOtherMeshMetadata(t *testing.T) {
	logger.Init(&logcfg.Config{Level: logcfg.LevelError, OutputPath: filepath.Join(t.TempDir(), "test.log")})
	for _, tc := range []struct{ mesh, otherMesh string }{
		{"nacos-local", "nacos-local/child"},
		{"nacos-local/child", "nacos-local"},
	} {
		t.Run(tc.mesh, func(t *testing.T) {
			providerStore, router, bus, lw, releaseList := startCatalogInformer(t, tc.mesh)
			awaitCatalogSignal(t, lw.listStarted, "list to start")
			stale := meshresource.NewServiceProviderMetadataResourceWithAttributes("stale", tc.mesh)
			require.NoError(t, providerStore.Add(stale))
			otherMesh := meshresource.NewServiceProviderMetadataResourceWithAttributes("other", tc.otherMesh)
			require.NoError(t, providerStore.Add(otherMesh))
			zk := ingestCatalogZKMetadata(t, bus, router)

			releaseList()
			awaitCatalogSignal(t, lw.watchStarted, "watch to start")
			// FIFO queues the sentinel after all initial-list reconciliation deltas.
			sentinel := meshresource.NewServiceProviderMetadataResourceWithAttributes("sentinel", tc.mesh)
			lw.watcher.Add(sentinel)
			var initial []events.Event
			for {
				e := nextCatalogEvent(t, bus.ch)
				if e.NewObj() != nil && e.NewObj().ResourceKey() == sentinel.ResourceKey() {
					break
				}
				initial = append(initial, e)
			}
			require.Contains(t, eventKeys(initial, cache.Deleted), stale.ResourceKey(), "relist must remove missing same-mesh metadata")
			require.NotContains(t, eventKeys(initial, cache.Deleted), zk.ResourceKey(), "another mesh must not be deleted by this list")
			require.NotContains(t, eventKeys(initial, cache.Deleted), otherMesh.ResourceKey(), "a mesh with a shared key prefix must remain")
			_, exists, err := providerStore.GetByKey(zk.ResourceKey())
			require.NoError(t, err)
			require.True(t, exists, "ZooKeeper provider metadata must remain in the shared store")
			_, exists, err = providerStore.GetByKey(otherMesh.ResourceKey())
			require.NoError(t, err)
			require.True(t, exists)
		})
	}
}

func TestDiscoveryInformerKeepsLateZKMetadata(t *testing.T) {
	logger.Init(&logcfg.Config{Level: logcfg.LevelError, OutputPath: filepath.Join(t.TempDir(), "test.log")})
	providerStore := memory.NewMemoryResourceStore(meshresource.ServiceProviderMetadataKind)
	lifecycleStore := memory.NewMemoryResourceStore(meshresource.LifecycleEventKind)
	require.NoError(t, providerStore.Init(nil))
	require.NoError(t, lifecycleStore.Init(nil))
	router := catalogTestRouter{meshresource.ServiceProviderMetadataKind: providerStore, meshresource.LifecycleEventKind: lifecycleStore}
	bus := &catalogTestBus{ch: make(chan events.Event, 20)}
	lw := &relistTestLW{
		watchStarted: make(chan int, 2),
		watchers:     [2]*watch.RaceFreeFakeWatcher{watch.NewRaceFreeFake(), watch.NewRaceFreeFake()},
	}
	oldRegistry := registry
	registry = newDiscoveryFactoryRegistry()
	t.Cleanup(func() { registry = oldRegistry })
	RegisterListWatcherFactory(&catalogTestFactory{lw: lw})
	informers, err := (&discoveryComponent{}).initInformers(&discoverycfg.Config{ID: "nacos-local", Type: "catalog-test"}, router, bus)
	require.NoError(t, err)
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() { defer close(done); informers[0].Run(stop) }()
	t.Cleanup(func() {
		close(stop)
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("timed out waiting for informer to stop")
		}
	})
	awaitWatch := func(expected int) {
		t.Helper()
		select {
		case n := <-lw.watchStarted:
			require.Equal(t, expected, n)
		case <-time.After(10 * time.Second):
			t.Fatal("timed out waiting for watch")
		}
	}
	awaitWatch(0)
	zk := ingestCatalogZKMetadata(t, bus, router)
	// A later relist with an empty catalog list would delete this object if
	// known-objects were not scoped to the informer's mesh.
	lw.watchers[0].Error(&metav1.Status{Status: metav1.StatusFailure, Reason: metav1.StatusReasonExpired, Code: 410})
	awaitWatch(1)
	sentinel := meshresource.NewServiceProviderMetadataResourceWithAttributes("sentinel", "nacos-local")
	lw.watchers[1].Add(sentinel)
	var afterRelist []events.Event
	for {
		e := nextCatalogEvent(t, bus.ch)
		if e.NewObj() != nil && e.NewObj().ResourceKey() == sentinel.ResourceKey() {
			break
		}
		afterRelist = append(afterRelist, e)
	}
	require.NotContains(t, eventKeys(afterRelist, cache.Deleted), zk.ResourceKey(), "late ZooKeeper metadata must not be deleted")
	_, exists, err := providerStore.GetByKey(zk.ResourceKey())
	require.NoError(t, err)
	require.True(t, exists, "late ZooKeeper metadata must remain in the shared store")
}

func TestDiscoveryInformerWatchAddUpdateDelete(t *testing.T) {
	logger.Init(&logcfg.Config{Level: logcfg.LevelError, OutputPath: filepath.Join(t.TempDir(), "test.log")})
	providerStore, _, bus, lw, releaseList := startCatalogInformer(t, "mesh")
	awaitCatalogSignal(t, lw.listStarted, "list to start")
	releaseList()
	awaitCatalogSignal(t, lw.watchStarted, "watch to start")

	obj := meshresource.NewServiceProviderMetadataResourceWithAttributes("provider", "mesh")
	lw.watcher.Add(obj)
	e := nextCatalogEvent(t, bus.ch)
	require.Equal(t, cache.Added, e.Type())
	require.Equal(t, obj.ResourceKey(), e.NewObj().ResourceKey())
	updated := obj.DeepCopyObject().(*meshresource.ServiceProviderMetadataResource)
	updated.Spec.ServiceName = "updated"
	lw.watcher.Modify(updated)
	e = nextCatalogEvent(t, bus.ch)
	require.Equal(t, cache.Updated, e.Type())
	require.Equal(t, "updated", e.NewObj().(*meshresource.ServiceProviderMetadataResource).Spec.ServiceName)
	stored, exists, err := providerStore.GetByKey(obj.ResourceKey())
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "updated", stored.(*meshresource.ServiceProviderMetadataResource).Spec.ServiceName)
	lw.watcher.Delete(updated)
	e = nextCatalogEvent(t, bus.ch)
	require.Equal(t, cache.Deleted, e.Type())
	require.Equal(t, obj.ResourceKey(), e.OldObj().ResourceKey())
	_, exists, err = providerStore.GetByKey(obj.ResourceKey())
	require.NoError(t, err)
	require.False(t, exists)
}

func TestDiscoveryInformerRelistDeletesStaleSameMesh(t *testing.T) {
	logger.Init(&logcfg.Config{Level: logcfg.LevelError, OutputPath: filepath.Join(t.TempDir(), "test.log")})
	providerStore := memory.NewMemoryResourceStore(meshresource.ServiceProviderMetadataKind)
	require.NoError(t, providerStore.Init(nil))
	lw := &relistTestLW{
		watchStarted: make(chan int, 2),
		watchers:     [2]*watch.RaceFreeFakeWatcher{watch.NewRaceFreeFake(), watch.NewRaceFreeFake()},
	}
	oldRegistry := registry
	registry = newDiscoveryFactoryRegistry()
	t.Cleanup(func() { registry = oldRegistry })
	RegisterListWatcherFactory(&catalogTestFactory{lw: lw})
	bus := &catalogTestBus{ch: make(chan events.Event, 20)}
	informers, err := (&discoveryComponent{}).initInformers(&discoverycfg.Config{ID: "mesh", Type: "catalog-test"}, catalogTestRouter{meshresource.ServiceProviderMetadataKind: providerStore}, bus)
	require.NoError(t, err)
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() { defer close(done); informers[0].Run(stop) }()
	t.Cleanup(func() {
		close(stop)
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("timed out waiting for informer to stop")
		}
	})
	awaitWatch := func(expected int) {
		t.Helper()
		select {
		case n := <-lw.watchStarted:
			require.Equal(t, expected, n)
		case <-time.After(10 * time.Second):
			t.Fatal("timed out waiting for watch")
		}
	}
	awaitWatch(0)
	stale := meshresource.NewServiceProviderMetadataResourceWithAttributes("stale", "mesh")
	lw.watchers[0].Add(stale)
	e := nextCatalogEvent(t, bus.ch)
	require.Equal(t, cache.Added, e.Type())
	require.Equal(t, stale.ResourceKey(), e.NewObj().ResourceKey())

	// An expired watch forces a fresh list that no longer contains the object.
	lw.watchers[0].Error(&metav1.Status{Status: metav1.StatusFailure, Reason: metav1.StatusReasonExpired, Code: 410})
	awaitWatch(1)
	e = nextCatalogEvent(t, bus.ch)
	require.Equal(t, cache.Deleted, e.Type())
	require.Equal(t, stale.ResourceKey(), e.OldObj().ResourceKey())
	_, exists, err := providerStore.GetByKey(stale.ResourceKey())
	require.NoError(t, err)
	require.False(t, exists)
}

func eventKeys(es []events.Event, typ cache.DeltaType) []string {
	var keys []string
	for _, e := range es {
		if e.Type() == typ && e.OldObj() != nil {
			keys = append(keys, e.OldObj().ResourceKey())
		}
	}
	return keys
}
