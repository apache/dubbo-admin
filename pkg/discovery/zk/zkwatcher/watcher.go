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

package zkwatcher

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dubbogo/go-zookeeper/zk"
)

// Connection is the subset of zk.Conn used by a recursive watch.
type Connection interface {
	GetW(string) ([]byte, *zk.Stat, *zk.Watcher, error)
	ChildrenW(string) ([]string, *zk.Stat, *zk.Watcher, error)
	RemoveWatcher(*zk.Watcher) bool
	State() zk.State
}

type EventType string

const (
	NodeCreated EventType = "NodeCreated"
	NodeDeleted EventType = "NodeDeleted"
	NodeChanged EventType = "NodeChanged"
)

type ZookeeperEvent struct {
	Type     EventType
	Path     string
	Data     string
	LeafNode bool
}

var errStopped = errors.New("zookeeper watcher stopped")

// RecursiveWatcher owns one snapshot and the watches installed while reading it.
// A topology change invalidates the snapshot so the reflector can reconcile deletes.
type RecursiveWatcher struct {
	conn      Connection
	basePath  string
	eventChan chan ZookeeperEvent
	errorChan chan error
	stopChan  chan struct{}
	doneChan  chan struct{}
	mu        sync.Mutex
	started   bool
	stopped   bool
	err       error
	workers   sync.WaitGroup
	snapshot  []ZookeeperEvent
}

func NewRecursiveWatcher(conn Connection, basePath string) *RecursiveWatcher {
	rw := &RecursiveWatcher{
		conn:      conn,
		basePath:  basePath,
		eventChan: make(chan ZookeeperEvent, 1000),
		errorChan: make(chan error, 1),
		stopChan:  make(chan struct{}),
		doneChan:  make(chan struct{}),
	}
	go func() {
		<-rw.stopChan
		rw.workers.Wait()
		close(rw.eventChan)
		close(rw.errorChan)
		close(rw.doneChan)
	}()
	return rw
}

// StartAsync reads the entire tree before returning, with watches already armed.
// Each RecursiveWatcher may only be started once.
func (rw *RecursiveWatcher) StartAsync() error {
	rw.mu.Lock()
	if rw.stopped {
		rw.mu.Unlock()
		return errStopped
	}
	if rw.started {
		rw.mu.Unlock()
		return errors.New("zookeeper watcher already started")
	}
	rw.started = true
	// Keep the counter positive until every initial worker has been registered.
	rw.workers.Add(1)
	rw.mu.Unlock()
	defer rw.workers.Done()

	if err := rw.watchPathRecursively(rw.basePath); err != nil {
		rw.cancel(err)
		return err
	}
	if err := rw.checkConnection(); err != nil {
		rw.cancel(err)
		return err
	}
	rw.workers.Add(1)
	go rw.monitorConnection()
	return rw.Err()
}

func (rw *RecursiveWatcher) Stop() {
	rw.cancel(nil)
	<-rw.doneChan
}

func (rw *RecursiveWatcher) cancel(err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	if rw.stopped {
		return
	}
	rw.stopped = true
	rw.err = err
	if err != nil {
		rw.errorChan <- err
	}
	close(rw.stopChan)
}

func (rw *RecursiveWatcher) Err() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	if rw.err != nil {
		return rw.err
	}
	if rw.stopped {
		return errStopped
	}
	return nil
}

// Snapshot must be called after StartAsync returns successfully.
func (rw *RecursiveWatcher) Snapshot() []ZookeeperEvent {
	return rw.snapshot
}

func (rw *RecursiveWatcher) EventChan() <-chan ZookeeperEvent {
	return rw.eventChan
}

func (rw *RecursiveWatcher) ErrorChan() <-chan error {
	return rw.errorChan
}

func (rw *RecursiveWatcher) watchPathRecursively(path string) error {
	if err := rw.Err(); err != nil {
		return err
	}
	data, stat, dataWatch, err := rw.conn.GetW(path)
	if err != nil {
		return fmt.Errorf("GetW %s: %w", path, err)
	}
	rw.snapshot = append(rw.snapshot, ZookeeperEvent{
		Type:     NodeCreated,
		Path:     path,
		Data:     string(data),
		LeafNode: stat.NumChildren == 0,
	})
	rw.workers.Add(1)
	go rw.watchDataChanges(path, dataWatch)

	if err := rw.Err(); err != nil {
		return err
	}
	children, _, childWatch, err := rw.conn.ChildrenW(path)
	if err != nil {
		return fmt.Errorf("ChildrenW %s: %w", path, err)
	}
	rw.workers.Add(1)
	go rw.watchChildrenChanges(path, childWatch)
	for _, child := range children {
		if err := rw.watchPathRecursively(path + "/" + child); err != nil {
			return err
		}
	}
	return nil
}

func (rw *RecursiveWatcher) watchDataChanges(path string, watcher *zk.Watcher) {
	defer rw.workers.Done()
	defer func() { rw.conn.RemoveWatcher(watcher) }()
	for {
		select {
		case <-rw.stopChan:
			return
		case event, ok := <-watcher.EvtCh:
			if !ok || event.Err != nil || event.Type != zk.EventNodeDataChanged {
				rw.cancel(fmt.Errorf("data watch invalidated at %s (open=%t): %v", path, ok, event))
				return
			}
		}
		rw.conn.RemoveWatcher(watcher)
		data, stat, next, err := rw.conn.GetW(path)
		if err != nil {
			rw.cancel(fmt.Errorf("GetW %s: %w", path, err))
			return
		}
		watcher = next
		select {
		case rw.eventChan <- ZookeeperEvent{
			Type: NodeChanged, Path: path, Data: string(data), LeafNode: stat.NumChildren == 0,
		}:
		case <-rw.stopChan:
			return
		}
	}
}

func (rw *RecursiveWatcher) watchChildrenChanges(path string, watcher *zk.Watcher) {
	defer rw.workers.Done()
	defer rw.conn.RemoveWatcher(watcher)
	select {
	case <-rw.stopChan:
	case event, ok := <-watcher.EvtCh:
		// Children watches are one-shot. Reconcile the whole tree instead of
		// missing descendants removed or recreated while rearming the watch.
		rw.cancel(fmt.Errorf("children watch invalidated at %s (open=%t): %v", path, ok, event))
	}
}

func (rw *RecursiveWatcher) monitorConnection() {
	defer rw.workers.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if err := rw.checkConnection(); err != nil {
			rw.cancel(err)
			return
		}
		select {
		case <-rw.stopChan:
			return
		case <-ticker.C:
		}
	}
}

func (rw *RecursiveWatcher) checkConnection() error {
	state := rw.conn.State()
	if state != zk.StateHasSession && state != zk.StateConnectedReadOnly {
		return fmt.Errorf("zookeeper connection is %s", state)
	}
	return nil
}
