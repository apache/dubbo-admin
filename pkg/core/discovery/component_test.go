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

func TestDiscoveryInformerKeepsOtherMeshMetadata(t *testing.T) {
	logger.Init(&logcfg.Config{Level: logcfg.LevelError, OutputPath: filepath.Join(t.TempDir(), "test.log")})
	for _, tc := range []struct {
		mesh, otherMesh string
		zkFirst         bool
	}{
		{"nacos-local", "nacos-local/child", true},
		{"nacos-local", "nacos-local/child", false},
		{"nacos-local/child", "nacos-local", true},
		{"nacos-local/child", "nacos-local", false},
	} {
		t.Run(fmt.Sprintf("mesh_%s_zookeeper_before_list_%t", tc.mesh, tc.zkFirst), func(t *testing.T) {
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
			informers, err := (&discoveryComponent{}).initInformers(&discoverycfg.Config{ID: tc.mesh, Type: "catalog-test"}, router, bus)
			require.NoError(t, err)
			stale := meshresource.NewServiceProviderMetadataResourceWithAttributes("stale", tc.mesh)
			require.NoError(t, providerStore.Add(stale))
			otherMesh := meshresource.NewServiceProviderMetadataResourceWithAttributes("other", tc.otherMesh)
			require.NoError(t, providerStore.Add(otherMesh))
			stop := make(chan struct{})
			done := make(chan struct{})
			go func() { defer close(done); informers[0].Run(stop) }()
			t.Cleanup(func() { close(stop); <-done })
			<-lw.listStarted

			ingestZK := func() coremodel.Resource {
				raw := meshresource.NewZKMetadataResourceWithAttributes("/dubbo/metadata/greet.GreetService/provider/app", "zk-local")
				raw.Spec = &meshproto.ZKMetadata{NodePath: raw.Name, NodeData: `{"canonicalName":"greet.GreetService","parameters":{"application":"app"}}`}
				err := subscriber.NewZKMetadataEventSubscriber(bus, router).ProcessEvent(events.NewResourceChangedEvent(cache.Added, nil, raw))
				require.NoError(t, err)
				e := nextCatalogEvent(t, bus.ch)
				require.Equal(t, cache.Added, e.Type())
				return e.NewObj()
			}
			var zk coremodel.Resource
			if tc.zkFirst {
				zk = ingestZK()
			}
			close(lw.allowList)
			<-lw.watchStarted
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
			if !tc.zkFirst {
				zk = ingestZK()
			}
			require.NotContains(t, eventKeys(initial, cache.Deleted), zk.ResourceKey(), "another mesh must not be deleted by this list")
			require.NotContains(t, eventKeys(initial, cache.Deleted), otherMesh.ResourceKey(), "a mesh with a shared key prefix must remain")
			_, exists, err := providerStore.GetByKey(zk.ResourceKey())
			require.NoError(t, err)
			require.True(t, exists, "ZooKeeper provider metadata must remain in the shared store")
			_, exists, err = providerStore.GetByKey(otherMesh.ResourceKey())
			require.NoError(t, err)
			require.True(t, exists)

			updated := sentinel.DeepCopyObject().(*meshresource.ServiceProviderMetadataResource)
			updated.Spec.ServiceName = "updated"
			lw.watcher.Modify(updated)
			e := nextCatalogEvent(t, bus.ch)
			require.Equal(t, cache.Updated, e.Type())
			require.Equal(t, "updated", e.NewObj().(*meshresource.ServiceProviderMetadataResource).Spec.ServiceName)
			stored, exists, err := providerStore.GetByKey(sentinel.ResourceKey())
			require.NoError(t, err)
			require.True(t, exists)
			require.Equal(t, "updated", stored.(*meshresource.ServiceProviderMetadataResource).Spec.ServiceName)
			deleteTarget := meshresource.NewServiceProviderMetadataResourceWithAttributes("delete-target", tc.mesh)
			lw.watcher.Add(deleteTarget)
			e = nextCatalogEvent(t, bus.ch)
			require.Equal(t, cache.Added, e.Type())
			require.Equal(t, deleteTarget.ResourceKey(), e.NewObj().ResourceKey())
			lw.watcher.Delete(deleteTarget)
			e = nextCatalogEvent(t, bus.ch)
			require.Equal(t, cache.Deleted, e.Type())
			require.Equal(t, deleteTarget.ResourceKey(), e.OldObj().ResourceKey())
			_, exists, err = providerStore.GetByKey(deleteTarget.ResourceKey())
			require.NoError(t, err)
			require.False(t, exists)
		})
	}
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
	t.Cleanup(func() { close(stop); <-done })
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
