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

package listerwatcher

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dubbogo/go-zookeeper/zk"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	discoverycfg "github.com/apache/dubbo-admin/pkg/config/discovery"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type fakeNode struct {
	data     string
	children []string
}

type fakeConnection struct {
	mu       sync.Mutex
	state    zk.State
	nodes    map[string]fakeNode
	watches  map[*zk.Watcher]bool
	getErr   error
	childErr error
	readHook func(string)
}

func newFakeConnection() *fakeConnection {
	return &fakeConnection{
		state: zk.StateHasSession,
		nodes: map[string]fakeNode{
			"/dubbo":      {children: []string{"old", "kept"}},
			"/dubbo/old":  {data: "old-data"},
			"/dubbo/kept": {data: "before"},
		},
		watches: make(map[*zk.Watcher]bool),
	}
}

func (c *fakeConnection) GetW(path string) ([]byte, *zk.Stat, *zk.Watcher, error) {
	c.mu.Lock()
	hook := c.readHook
	c.mu.Unlock()
	if hook != nil {
		hook(path)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.getErr != nil {
		return nil, nil, nil, c.getErr
	}
	node, ok := c.nodes[path]
	if !ok {
		return nil, nil, nil, zk.ErrNoNode
	}
	w := c.addWatch(path, zk.WatchTypeData)
	return []byte(node.data), &zk.Stat{NumChildren: int32(len(node.children))}, w, nil
}

func (c *fakeConnection) ChildrenW(path string) ([]string, *zk.Stat, *zk.Watcher, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.childErr != nil {
		return nil, nil, nil, c.childErr
	}
	node, ok := c.nodes[path]
	if !ok {
		return nil, nil, nil, zk.ErrNoNode
	}
	return append([]string(nil), node.children...), &zk.Stat{}, c.addWatch(path, zk.WatchTypeChild), nil
}

func (c *fakeConnection) addWatch(path string, typ zk.WatchType) *zk.Watcher {
	w := &zk.Watcher{Wpt: zk.WatchPathType{Path: path, WType: typ}, EvtCh: make(chan zk.Event, 1)}
	c.watches[w] = true
	return w
}

func (c *fakeConnection) RemoveWatcher(w *zk.Watcher) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.watches[w]
	delete(c.watches, w)
	return ok
}

func (c *fakeConnection) State() zk.State {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

func (c *fakeConnection) fire(path string, typ zk.WatchType, event *zk.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for w := range c.watches {
		if w.Wpt.Path == path && w.Wpt.WType == typ {
			delete(c.watches, w)
			if event != nil {
				w.EvtCh <- *event
			}
			close(w.EvtCh)
		}
	}
}

func newTestListerWatcher(conn *fakeConnection) *ListerWatcher[coremodel.Resource] {
	return &ListerWatcher[coremodel.Resource]{
		basePath:       "/dubbo",
		rk:             meshresource.ZKMetadataKind,
		conn:           conn,
		cfg:            &discoverycfg.Config{ID: "test"},
		newResListFunc: meshresource.NewZKMetadataResourceList,
		toUpsertResourceFunc: func(mesh, path, data string) coremodel.Resource {
			if path == "/dubbo" {
				return nil
			}
			r := meshresource.NewZKMetadataResourceWithAttributes(path, mesh)
			r.Annotations = map[string]string{"data": data}
			return r
		},
		toDeleteResourceFunc: func(mesh, path string) coremodel.Resource {
			return meshresource.NewZKMetadataResourceWithAttributes(path, mesh)
		},
	}
}

func receive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value, ok := <-ch:
		require.True(t, ok, "channel closed before expected value")
		return value
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for channel")
		var zero T
		return zero
	}
}

func snapshot(store cache.Store) map[string]string {
	items := make(map[string]string)
	for _, obj := range store.List() {
		r := obj.(*meshresource.ZKMetadataResource)
		items[r.Name] = r.Annotations["data"]
	}
	return items
}

func TestReflectorRelistsAfterWatchLossBeforeReportingHealthy(t *testing.T) {
	for _, loss := range []string{"closed-data", "closed-children", "not-watching", "data-error", "children-changed", "disconnected"} {
		t.Run(loss, func(t *testing.T) {
			conn := newFakeConnection()
			lw := newTestListerWatcher(conn)
			t.Cleanup(lw.Stop)
			store := cache.NewStore(func(obj interface{}) (string, error) {
				return obj.(coremodel.Resource).ResourceKey(), nil
			})
			var healthy atomic.Bool
			synced := make(chan map[string]string, 8)
			lw.SetWatchHealthHandler(func(err error) {
				healthy.Store(err == nil)
				if err == nil {
					synced <- snapshot(store)
				}
			})
			// Use the pinned reflector's public API with its default options.
			reflector := cache.NewReflector(lw, &meshresource.ZKMetadataResource{}, store, 0)
			stop := make(chan struct{})
			defer close(stop)
			first := make(chan error, 1)
			go func() { first <- reflector.ListAndWatch(stop) }()
			require.Equal(t, map[string]string{"/dubbo/old": "old-data", "/dubbo/kept": "before"}, receive(t, synced))

			switch loss {
			case "closed-data":
				conn.fire("/dubbo/kept", zk.WatchTypeData, nil)
			case "closed-children":
				conn.fire("/dubbo", zk.WatchTypeChild, nil)
			case "not-watching":
				conn.fire("/dubbo/kept", zk.WatchTypeData, &zk.Event{Type: zk.EventNotWatching, Err: zk.ErrSessionExpired})
			case "data-error":
				conn.mu.Lock()
				conn.getErr = zk.ErrConnectionClosed
				conn.mu.Unlock()
				conn.fire("/dubbo/kept", zk.WatchTypeData, &zk.Event{Type: zk.EventNodeDataChanged})
			case "children-changed":
				conn.fire("/dubbo", zk.WatchTypeChild, &zk.Event{Type: zk.EventNodeChildrenChanged})
			case "disconnected":
				conn.mu.Lock()
				conn.state = zk.StateDisconnected
				conn.mu.Unlock()
			}
			require.NoError(t, receive(t, first))
			require.False(t, healthy.Load())

			entered, resume := make(chan struct{}), make(chan struct{})
			var once sync.Once
			var resumeOnce sync.Once
			defer resumeOnce.Do(func() { close(resume) })
			conn.mu.Lock()
			conn.getErr = nil
			conn.state = zk.StateHasSession
			conn.nodes = map[string]fakeNode{
				"/dubbo":      {children: []string{"new", "kept"}},
				"/dubbo/new":  {data: "new-data"},
				"/dubbo/kept": {data: "after"},
			}
			conn.readHook = func(path string) {
				if path == "/dubbo/kept" {
					once.Do(func() { close(entered) })
					<-resume
				}
			}
			conn.mu.Unlock()
			second := make(chan error, 1)
			go func() { second <- reflector.ListAndWatch(stop) }()
			select {
			case <-entered:
			case <-time.After(3 * time.Second):
				t.Fatal("reflector did not start a full relist")
			}
			// A connected session with an incomplete relist must stay unhealthy.
			require.False(t, healthy.Load())
			require.Equal(t, map[string]string{"/dubbo/old": "old-data", "/dubbo/kept": "before"}, snapshot(store))
			resumeOnce.Do(func() { close(resume) })
			require.Equal(t, map[string]string{"/dubbo/new": "new-data", "/dubbo/kept": "after"}, receive(t, synced))
			require.True(t, healthy.Load())
			lw.Stop()
			require.NoError(t, receive(t, second))
		})
	}
}

func TestListFailureIsUnhealthyAndRecoveryRequiresFreshList(t *testing.T) {
	for _, method := range []string{"GetW", "ChildrenW"} {
		t.Run(method, func(t *testing.T) {
			conn := newFakeConnection()
			lw := newTestListerWatcher(conn)
			t.Cleanup(lw.Stop)
			var healthy atomic.Bool
			lw.SetWatchHealthHandler(func(err error) { healthy.Store(err == nil) })
			if method == "GetW" {
				conn.getErr = zk.ErrConnectionClosed
			} else {
				conn.childErr = zk.ErrConnectionClosed
			}
			_, err := lw.List(metav1.ListOptions{})
			require.Error(t, err)
			require.False(t, healthy.Load())
			_, err = lw.Watch(metav1.ListOptions{})
			require.True(t, apierrors.IsResourceExpired(err))
			conn.mu.Lock()
			conn.getErr, conn.childErr = nil, nil
			conn.mu.Unlock()
			_, err = lw.List(metav1.ListOptions{})
			require.NoError(t, err)
			require.False(t, healthy.Load())
			stream, err := lw.Watch(metav1.ListOptions{})
			require.NoError(t, err)
			require.True(t, healthy.Load())
			stream.Stop()
			require.False(t, healthy.Load())
			conn.mu.Lock()
			require.Empty(t, conn.watches)
			conn.mu.Unlock()
		})
	}
}

func TestIncrementalDataChangeAndWatchRearmFailure(t *testing.T) {
	conn := newFakeConnection()
	lw := newTestListerWatcher(conn)
	t.Cleanup(lw.Stop)
	_, err := lw.List(metav1.ListOptions{})
	require.NoError(t, err)
	stream, err := lw.Watch(metav1.ListOptions{})
	require.NoError(t, err)
	conn.mu.Lock()
	conn.nodes["/dubbo/kept"] = fakeNode{data: "updated"}
	conn.mu.Unlock()
	conn.fire("/dubbo/kept", zk.WatchTypeData, &zk.Event{Type: zk.EventNodeDataChanged})
	event := receive(t, stream.ResultChan())
	require.Equal(t, watch.Modified, event.Type)
	require.Equal(t, "updated", event.Object.(*meshresource.ZKMetadataResource).Annotations["data"])
	conn.fire("/dubbo/kept", zk.WatchTypeData, nil)
	event = receive(t, stream.ResultChan())
	require.Equal(t, watch.Error, event.Type)
	require.True(t, apierrors.IsResourceExpired(apierrors.FromObject(event.Object)))
}

func TestWatchStopIsIndependentAndRepeatable(t *testing.T) {
	conn := newFakeConnection()
	lw := newTestListerWatcher(conn)
	require.NotPanics(t, func() { lw.Stop(); lw.Stop() })
	var healthy atomic.Bool
	lw.SetWatchHealthHandler(func(err error) { healthy.Store(err == nil) })
	start := func() watch.Interface {
		_, err := lw.List(metav1.ListOptions{})
		require.NoError(t, err)
		stream, err := lw.Watch(metav1.ListOptions{})
		require.NoError(t, err)
		return stream
	}
	old := start()
	var stops sync.WaitGroup
	for i := 0; i < 20; i++ {
		stops.Add(1)
		go func() { defer stops.Done(); old.Stop() }()
	}
	stops.Wait()
	_, ok := <-old.ResultChan()
	require.False(t, ok)
	stream := start()
	t.Cleanup(stream.Stop)
	require.NotEqual(t, old.ResultChan(), stream.ResultChan())
	old.Stop()
	require.True(t, healthy.Load())
	conn.fire("/dubbo/kept", zk.WatchTypeData, &zk.Event{Type: zk.EventNodeDataChanged})
	require.Equal(t, watch.Modified, receive(t, stream.ResultChan()).Type)
}

func TestWatchCannotReportHealthyAfterInvalidationBetweenListAndWatch(t *testing.T) {
	conn := newFakeConnection()
	lw := newTestListerWatcher(conn)
	t.Cleanup(lw.Stop)
	var healthy atomic.Bool
	lw.SetWatchHealthHandler(func(err error) { healthy.Store(err == nil) })
	_, err := lw.List(metav1.ListOptions{})
	require.NoError(t, err)
	conn.fire("/dubbo", zk.WatchTypeChild, &zk.Event{Type: zk.EventNodeChildrenChanged})
	require.Eventually(t, func() bool { return lw.stream.watcher.Err() != nil }, time.Second, time.Millisecond)
	_, err = lw.Watch(metav1.ListOptions{})
	require.True(t, apierrors.IsResourceExpired(err))
	require.False(t, healthy.Load())
}

func TestReflectorAutomaticallyRelistsAfterClosedWatch(t *testing.T) {
	conn := newFakeConnection()
	lw := newTestListerWatcher(conn)
	store := cache.NewStore(func(obj interface{}) (string, error) {
		return obj.(coremodel.Resource).ResourceKey(), nil
	})
	synced := make(chan map[string]string, 8)
	lw.SetWatchHealthHandler(func(err error) {
		if err == nil {
			synced <- snapshot(store)
		}
	})
	stop, done := make(chan struct{}), make(chan struct{})
	reflector := cache.NewReflector(lw, &meshresource.ZKMetadataResource{}, store, 0)
	go func() {
		defer close(done)
		reflector.Run(stop)
	}()
	t.Cleanup(func() {
		close(stop)
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("reflector did not stop")
		}
		lw.Stop()
	})
	receive(t, synced)
	// Consume a real update first: a plain close after an event normally just
	// reopens Watch in client-go instead of returning to List.
	conn.mu.Lock()
	conn.nodes["/dubbo/kept"] = fakeNode{data: "updated"}
	conn.mu.Unlock()
	conn.fire("/dubbo/kept", zk.WatchTypeData, &zk.Event{Type: zk.EventNodeDataChanged})
	require.Eventually(t, func() bool {
		return snapshot(store)["/dubbo/kept"] == "updated"
	}, time.Second, time.Millisecond)
	conn.mu.Lock()
	conn.nodes["/dubbo"] = fakeNode{children: []string{"kept"}}
	delete(conn.nodes, "/dubbo/old")
	conn.mu.Unlock()
	conn.fire("/dubbo/kept", zk.WatchTypeData, nil)
	require.Equal(t, map[string]string{"/dubbo/kept": "updated"}, receive(t, synced))
}

func TestStopWithFullResultChannel(t *testing.T) {
	conn := newFakeConnection()
	lw := newTestListerWatcher(conn)
	t.Cleanup(lw.Stop)
	converted := make(chan struct{})
	upsert := lw.toUpsertResourceFunc
	lw.toUpsertResourceFunc = func(mesh, path, data string) coremodel.Resource {
		if data == "blocked" {
			close(converted)
		}
		return upsert(mesh, path, data)
	}
	_, err := lw.List(metav1.ListOptions{})
	require.NoError(t, err)
	stream, err := lw.Watch(metav1.ListOptions{})
	require.NoError(t, err)
	result := stream.(*resourceWatch).resultChan
	for i := 0; i < cap(result); i++ {
		result <- watch.Event{Type: watch.Bookmark}
	}
	conn.mu.Lock()
	conn.nodes["/dubbo/kept"] = fakeNode{data: "blocked"}
	conn.mu.Unlock()
	conn.fire("/dubbo/kept", zk.WatchTypeData, &zk.Event{Type: zk.EventNodeDataChanged})
	select {
	case <-converted:
	case <-time.After(time.Second):
		t.Fatal("watch did not reach the blocked send")
	}
	stopped := make(chan bool, 1)
	go func() {
		stream.Stop()
		stream.Stop()
		stopped <- true
	}()
	require.True(t, receive(t, stopped))
	conn.mu.Lock()
	require.Empty(t, conn.watches)
	conn.mu.Unlock()
	for range stream.ResultChan() {
	}
}

func TestStopDuringSnapshotWaitsForProducerAndAllowsRecovery(t *testing.T) {
	conn := newFakeConnection()
	lw := newTestListerWatcher(conn)
	t.Cleanup(lw.Stop)
	entered, resume := make(chan struct{}), make(chan struct{})
	var resumeOnce sync.Once
	defer resumeOnce.Do(func() { close(resume) })
	conn.readHook = func(path string) {
		if path == "/dubbo" {
			close(entered)
			<-resume
		}
	}
	listed := make(chan error, 1)
	go func() {
		_, err := lw.List(metav1.ListOptions{})
		listed <- err
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("snapshot did not begin")
	}
	stopped := make(chan bool, 1)
	go func() {
		lw.Stop()
		stopped <- true
	}()
	require.Eventually(t, func() bool { return lw.stream.watcher.Err() != nil }, time.Second, time.Millisecond)
	resumeOnce.Do(func() { close(resume) })
	require.Error(t, receive(t, listed))
	require.True(t, receive(t, stopped))
	conn.mu.Lock()
	require.Empty(t, conn.watches)
	conn.readHook = nil
	conn.mu.Unlock()
	_, err := lw.List(metav1.ListOptions{})
	require.NoError(t, err)
	stream, err := lw.Watch(metav1.ListOptions{})
	require.NoError(t, err)
	stream.Stop()
}

func TestRecursiveListPropagatesDescendantFailure(t *testing.T) {
	for _, method := range []string{"GetW", "ChildrenW"} {
		t.Run(method, func(t *testing.T) {
			conn := newFakeConnection()
			lw := newTestListerWatcher(conn)
			t.Cleanup(lw.Stop)
			conn.readHook = func(path string) {
				if path == "/dubbo/old" {
					conn.mu.Lock()
					if method == "GetW" {
						conn.getErr = zk.ErrConnectionClosed
					} else {
						conn.childErr = zk.ErrConnectionClosed
					}
					conn.mu.Unlock()
				}
			}
			_, err := lw.List(metav1.ListOptions{})
			require.ErrorIs(t, err, zk.ErrConnectionClosed)
			conn.mu.Lock()
			require.Empty(t, conn.watches)
			conn.mu.Unlock()
		})
	}
}
