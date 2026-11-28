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
	"fmt"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	discoverycfg "github.com/apache/dubbo-admin/pkg/config/discovery"
	"github.com/apache/dubbo-admin/pkg/core/clients"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/discovery/zk/zkwatcher"
)

type ToUpsertResourceFunc func(mesh, nodePath, nodeData string) coremodel.Resource

type ToDeleteResourceFunc func(mesh, nodePath string) coremodel.Resource

type ListerWatcher[T coremodel.Resource] struct {
	basePath             string
	rk                   coremodel.ResourceKind
	conn                 zkwatcher.Connection
	cfg                  *discoverycfg.Config
	toUpsertResourceFunc ToUpsertResourceFunc
	toDeleteResourceFunc ToDeleteResourceFunc
	newResourceFunc      coremodel.NewResourceFunc
	newResListFunc       coremodel.NewResourceListFunc
	listMu               sync.Mutex
	mu                   sync.Mutex
	stream               *resourceWatch
	pending              *resourceWatch
	healthMu             sync.RWMutex
	watchHealth          func(error)
}

func NewListerWatcher(
	rk coremodel.ResourceKind,
	toResourceFunc ToUpsertResourceFunc,
	toDeleteResourceFunc ToDeleteResourceFunc,
	basePath string,
	cfg *discoverycfg.Config) (*ListerWatcher[coremodel.Resource], error) {
	newResourceFunc, err := coremodel.ResourceSchemaRegistry().NewResourceFunc(rk)
	if err != nil {
		return nil, err
	}
	newResListFunc, err := coremodel.ResourceSchemaRegistry().NewResourceListFunc(rk)
	if err != nil {
		return nil, err
	}
	conn, err := clients.NewZKConnection(cfg.Address.Registry)
	if err != nil {
		return nil, err
	}
	return &ListerWatcher[coremodel.Resource]{
		rk:                   rk,
		toUpsertResourceFunc: toResourceFunc,
		toDeleteResourceFunc: toDeleteResourceFunc,
		basePath:             basePath,
		conn:                 conn,
		cfg:                  cfg,
		newResourceFunc:      newResourceFunc,
		newResListFunc:       newResListFunc,
	}, nil
}

func (lw *ListerWatcher[T]) List(_ metav1.ListOptions) (k8sruntime.Object, error) {
	lw.listMu.Lock()
	defer lw.listMu.Unlock()
	lw.Stop()
	lw.reportWatchHealth(fmt.Errorf("zookeeper snapshot is being reconciled for %s", lw.basePath))

	stream := &resourceWatch{
		watcher:    zkwatcher.NewRecursiveWatcher(lw.conn, lw.basePath),
		resultChan: make(chan watch.Event, 1000),
		stopChan:   make(chan struct{}),
		doneChan:   make(chan struct{}),
		health:     lw.reportWatchHealth,
		toEvent:    lw.toWatchEvent,
	}
	lw.mu.Lock()
	lw.stream, lw.pending = stream, nil
	lw.mu.Unlock()
	go stream.run()
	if err := stream.watcher.StartAsync(); err != nil {
		stream.Stop()
		lw.reportWatchHealth(err)
		return nil, err
	}

	items := make([]coremodel.Resource, 0)
	for _, event := range stream.watcher.Snapshot() {
		if res := lw.toUpsertResourceFunc(lw.cfg.ID, event.Path, event.Data); res != nil {
			items = append(items, res)
		}
	}
	if err := stream.watcher.Err(); err != nil {
		stream.Stop()
		lw.reportWatchHealth(err)
		return nil, err
	}
	list := lw.newResListFunc()
	list.SetItems(items)
	lw.mu.Lock()
	lw.pending = stream
	lw.mu.Unlock()
	return list, nil
}

func (lw *ListerWatcher[T]) Watch(_ metav1.ListOptions) (watch.Interface, error) {
	lw.mu.Lock()
	stream := lw.pending
	lw.pending = nil
	lw.mu.Unlock()
	if stream == nil {
		err := apierrors.NewResourceExpired("zookeeper watch requires a fresh List")
		lw.reportWatchHealth(err)
		return nil, err
	}
	// Reflector has replaced its store with List's complete snapshot before Watch.
	// A connection/session transition alone must never restore health.
	if err := stream.reportReady(); err != nil {
		stream.Stop()
		return nil, apierrors.NewResourceExpired(err.Error())
	}
	return stream, nil
}

func (lw *ListerWatcher[T]) SetWatchHealthHandler(handler func(error)) {
	lw.healthMu.Lock()
	defer lw.healthMu.Unlock()
	lw.watchHealth = handler
}

func (lw *ListerWatcher[T]) reportWatchHealth(err error) {
	lw.healthMu.RLock()
	handler := lw.watchHealth
	lw.healthMu.RUnlock()
	if handler != nil {
		handler(err)
	}
}

func (lw *ListerWatcher[T]) toWatchEvent(event zkwatcher.ZookeeperEvent) *watch.Event {
	var res coremodel.Resource
	var typ watch.EventType
	switch event.Type {
	case zkwatcher.NodeCreated:
		res, typ = lw.toUpsertResourceFunc(lw.cfg.ID, event.Path, event.Data), watch.Added
	case zkwatcher.NodeChanged:
		res, typ = lw.toUpsertResourceFunc(lw.cfg.ID, event.Path, event.Data), watch.Modified
	case zkwatcher.NodeDeleted:
		res, typ = lw.toDeleteResourceFunc(lw.cfg.ID, event.Path), watch.Deleted
	}
	if res == nil {
		return nil
	}
	return &watch.Event{Type: typ, Object: res}
}

func (lw *ListerWatcher[T]) ResourceKind() coremodel.ResourceKind {
	return lw.rk
}

func (lw *ListerWatcher[T]) TransformFunc() cache.TransformFunc {
	return nil
}

// Stop stops the current watch, leaving the connection available for the next List.
func (lw *ListerWatcher[T]) Stop() {
	lw.mu.Lock()
	stream := lw.stream
	lw.mu.Unlock()
	if stream != nil {
		stream.Stop()
	}
}

// Close releases the shared connection after the informer has stopped.
func (lw *ListerWatcher[T]) Close() {
	lw.listMu.Lock()
	defer lw.listMu.Unlock()
	lw.Stop()
	if conn, ok := lw.conn.(interface{ Close() }); ok {
		conn.Close()
	}
}

func (lw *ListerWatcher[T]) ResultChan() <-chan watch.Event {
	lw.mu.Lock()
	defer lw.mu.Unlock()
	if lw.stream == nil {
		return nil
	}
	return lw.stream.ResultChan()
}

// resourceWatch is a single watch generation. Only run sends to/closes resultChan.
type resourceWatch struct {
	watcher    *zkwatcher.RecursiveWatcher
	resultChan chan watch.Event
	stopChan   chan struct{}
	doneChan   chan struct{}
	stopOnce   sync.Once
	healthMu   sync.Mutex
	health     func(error)
	toEvent    func(zkwatcher.ZookeeperEvent) *watch.Event
}

func (w *resourceWatch) Stop() {
	w.stopOnce.Do(func() { close(w.stopChan) })
	// Cancel producers too, including a snapshot still being read by List.
	w.watcher.Stop()
	<-w.doneChan
}

func (w *resourceWatch) ResultChan() <-chan watch.Event {
	return w.resultChan
}

func (w *resourceWatch) reportReady() error {
	w.healthMu.Lock()
	defer w.healthMu.Unlock()
	err := w.watcher.Err()
	w.health(err)
	return err
}

func (w *resourceWatch) run() {
	defer func() {
		w.healthMu.Lock()
		w.health(fmt.Errorf("zookeeper watch stopped"))
		w.healthMu.Unlock()
		w.watcher.Stop()
		close(w.resultChan)
		close(w.doneChan)
	}()
	for {
		select {
		case <-w.stopChan:
			return
		case err, ok := <-w.watcher.ErrorChan():
			if !ok {
				err = w.watcher.Err()
			}
			w.invalidate(err)
			return
		case event, ok := <-w.watcher.EventChan():
			if !ok {
				w.invalidate(w.watcher.Err())
				return
			}
			if err := w.watcher.Err(); err != nil {
				w.invalidate(err)
				return
			}
			if result := w.toEvent(event); result != nil {
				select {
				case w.resultChan <- *result:
				case <-w.stopChan:
					return
				case err := <-w.watcher.ErrorChan():
					if err == nil {
						err = w.watcher.Err()
					}
					w.invalidate(err)
					return
				}
			}
		}
	}
}

func (w *resourceWatch) invalidate(err error) {
	w.healthMu.Lock()
	w.health(err)
	w.healthMu.Unlock()
	// A plain close can restart Watch without List in client-go. Expired forces
	// a new List/Replace, including deletions missed while disconnected.
	status := apierrors.NewResourceExpired(err.Error()).ErrStatus
	select {
	case w.resultChan <- watch.Event{Type: watch.Error, Object: &status}:
	case <-w.stopChan:
	}
}
