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
	"sync"
	"time"

	"github.com/dubbogo/go-zookeeper/zk"
	set "github.com/duke-git/lancet/v2/datastructure/set"

	"github.com/apache/dubbo-admin/pkg/core/logger"
)

type EventType string

const (
	NodeCreated EventType = "NodeCreated"
	NodeDeleted EventType = "NodeDeleted"
	NodeChanged EventType = "NodeChanged"
)

// ZookeeperEvent represents a zookeeper event
type ZookeeperEvent struct {
	Type EventType
	// node path
	Path string
	// node data
	Data string
	// whether it is a leaf node
	LeafNode bool
}

// RecursiveWatcher recursive watcher
type RecursiveWatcher struct {
	conn      *zk.Conn
	basePath  string
	eventChan chan ZookeeperEvent
	stopChan  chan struct{}
	mu        sync.Mutex
}

// NewRecursiveWatcher create a new recursive watcher
func NewRecursiveWatcher(conn *zk.Conn, basePath string) *RecursiveWatcher {
	return &RecursiveWatcher{
		conn:      conn,
		basePath:  basePath,
		eventChan: make(chan ZookeeperEvent, 1000),
		stopChan:  make(chan struct{}),
	}
}

// StartAsync begin watching
func (rw *RecursiveWatcher) StartAsync() error {
	logger.Infof("Start watching path: %s", rw.basePath)
	// Recursively watch the initial path
	return rw.watchPathRecursively(rw.basePath)
}

// Stop watching
func (rw *RecursiveWatcher) Stop() {
	logger.Infof("Stop watching path: %s", rw.basePath)
	close(rw.stopChan)
	close(rw.eventChan)
}

// EventChan return event channel
func (rw *RecursiveWatcher) EventChan() <-chan ZookeeperEvent {
	return rw.eventChan
}

// watchPathRecursively watch path recursively
func (rw *RecursiveWatcher) watchPathRecursively(path string) error {
	// Watch data changes for the current path
	go rw.watchDataChanges(path)

	// Get children of current path and watch
	children, _, err := rw.conn.Children(path)
	if err != nil {
		logger.Errorf("Failed to get children of path: %s, cause: %v", path, err)
		return err
	}
	// Watch children list changes
	go rw.watchChildrenChanges(path)

	// Recursively watch child nodes
	for _, child := range children {
		childPath := path + "/" + child
		if err := rw.watchPathRecursively(childPath); err != nil {
			logger.Errorf("Failed to watch subpath %s: %v", childPath, err)
		}
	}
	return nil
}

// watchDataChanges watch node data changes
func (rw *RecursiveWatcher) watchDataChanges(path string) {
	logger.Debugf("Watching data changes for path: %s", path)
	for {

		select {
		case <-rw.stopChan:
			return
		default:
		}

		// Get data and set up watch
		data, stat, watcher, err := rw.conn.GetW(path)
		if err != nil {
			logger.Errorf("Failed to watch data changes for path %s, cause: %v", path, err)
			time.Sleep(time.Second)
			continue
		}
		rw.eventChan <- ZookeeperEvent{
			Type:     NodeCreated,
			Path:     path,
			Data:     string(data),
			LeafNode: stat.NumChildren == 0,
		}

		select {
		case event := <-watcher.EvtCh:
			if event.Type == zk.EventNodeDataChanged {
				logger.Debugf("node data changed: %s", path)

				// Re-watch data changes
				go rw.watchDataChanges(path)
				return
			} else if event.Type == zk.EventNodeDeleted {
				// Node deleted, stop watching
				logger.Debugf("node deleted: %s", path)
				rw.eventChan <- ZookeeperEvent{
					Type:     NodeDeleted,
					Path:     path,
					LeafNode: stat.NumChildren == 0,
				}
				return
			}
		case <-rw.stopChan:
			return
		}

		// If node exists but no changes, continue watching
		if stat != nil {
			continue
		}
	}
}

// watchChildrenChanges watch child node changes
func (rw *RecursiveWatcher) watchChildrenChanges(path string) {
	logger.Debugf("Watching child node changes for path: %s", path)
	for {
		// Check if stopped
		select {
		case <-rw.stopChan:
			return
		default:
		}

		// Get child nodes and set up watch
		children, stat, watcher, err := rw.conn.ChildrenW(path)
		if err != nil {
			logger.Errorf("Failed to watch child node changes for path %s: %v", path, err)
			time.Sleep(time.Second)
			continue
		}

		select {
		case event := <-watcher.EvtCh:
			if event.Type == zk.EventNodeChildrenChanged {
				logger.Debugf("Child node list changed: %s", path)

				// Get current child nodes
				newChildren, _, err := rw.conn.Children(path)
				if err != nil {
					logger.Debugf("Failed to get new child nodes: %v", err)
					continue
				}
				// Calculate added and deleted child nodes
				newChildSet := set.FromSlice(newChildren)
				oldChildSet := set.FromSlice(children)
				addedChildren := newChildSet.Minus(oldChildSet)
				deletedChildren := oldChildSet.Minus(newChildSet)

				// watch newly added child nodes
				for _, c := range addedChildren.ToSlice() {
					childPath := path + "/" + c
					logger.Debugf("New child node added: %s", childPath)
					// Recursively watch new node
					go func() {
						err := rw.watchPathRecursively(childPath)
						if err != nil {
							logger.Errorf("Failed to watch subpath %s: %v", childPath, err)
						}
					}()
				}

				// Check for deleted child nodes
				for _, c := range deletedChildren.ToSlice() {
					logger.Debugf("child node %s of %s deleted", c, path)
				}

				// Re-watch child node changes
				go rw.watchChildrenChanges(path)
				return
			} else if event.Type == zk.EventNodeDeleted {
				// Parent node deleted, stop watching
				logger.Debugf("parent node %s deleted, stop watching children changes", path)
				return
			}
		case <-rw.stopChan:
			return
		}

		// If node exists but no changes, continue watching
		if stat != nil {
			continue
		}
	}
}
