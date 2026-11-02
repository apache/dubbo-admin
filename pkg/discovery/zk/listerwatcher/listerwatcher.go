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

	"github.com/dubbogo/go-zookeeper/zk"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	discoverycfg "github.com/apache/dubbo-admin/pkg/config/discovery"
	"github.com/apache/dubbo-admin/pkg/core/clients"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/discovery/zk/zkwatcher"
)

type ToUpsertResourceFunc func(mesh, nodePath, nodeData string) coremodel.Resource

type ToDeleteResourceFunc func(mesh, nodePath string) coremodel.Resource

type ListerWatcher[T coremodel.Resource] struct {
	basePath             string
	rk                   coremodel.ResourceKind
	conn                 *zk.Conn
	cfg                  *discoverycfg.Config
	zkAddress            string
	toUpsertResourceFunc ToUpsertResourceFunc
	toDeleteResourceFunc ToDeleteResourceFunc
	newResourceFunc      coremodel.NewResourceFunc
	newResListFunc       coremodel.NewResourceListFunc
	watcher              *zkwatcher.RecursiveWatcher
	resultChan           chan watch.Event
	stopChan             chan struct{}
}

func NewListerWatcher(
	rk coremodel.ResourceKind,
	toResourceFunc ToUpsertResourceFunc,
	toDeleteResourceFunc ToDeleteResourceFunc,
	basePath string,
	cfg *discoverycfg.Config) (*ListerWatcher[coremodel.Resource], error) {
	return NewListerWatcherWithAddress(rk, toResourceFunc, toDeleteResourceFunc, basePath, cfg, cfg.Address.Registry)
}

// NewListerWatcherWithAddress is used when a resource lives in a dedicated
// config-center endpoint instead of the service registry endpoint.
func NewListerWatcherWithAddress(
	rk coremodel.ResourceKind,
	toResourceFunc ToUpsertResourceFunc,
	toDeleteResourceFunc ToDeleteResourceFunc,
	basePath string,
	cfg *discoverycfg.Config,
	zkAddress string) (*ListerWatcher[coremodel.Resource], error) {
	if zkAddress == "" {
		zkAddress = cfg.Address.Registry
	}
	newResourceFunc, err := coremodel.ResourceSchemaRegistry().NewResourceFunc(rk)
	if err != nil {
		return nil, err
	}
	newResListFunc, err := coremodel.ResourceSchemaRegistry().NewResourceListFunc(rk)
	if err != nil {
		return nil, err
	}
	conn, err := clients.NewZKConnection(zkAddress)
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
		zkAddress:            zkAddress,
		newResourceFunc:      newResourceFunc,
		newResListFunc:       newResListFunc,
		resultChan:           make(chan watch.Event, 1000),
		stopChan:             make(chan struct{}),
	}, nil
}

func (lw *ListerWatcher[T]) List(_ metav1.ListOptions) (k8sruntime.Object, error) {
	resList, err := lw.listRecursively(lw.basePath)
	if err != nil {
		logger.Errorf("list all %s under path %s in zk %s failed, cause: %v", lw.rk, lw.basePath, lw.zkAddr(), err)
		return nil, err
	}
	resListObj := lw.newResListFunc()
	resListObj.SetItems(resList)
	return resListObj, nil
}

func (lw *ListerWatcher[T]) listRecursively(path string) ([]coremodel.Resource, error) {
	resList := make([]coremodel.Resource, 0)
	nodeData, _, err := lw.conn.Get(path)
	if err != nil {
		errStr := fmt.Sprintf("get %s from zk failed, path: %s, addr: %s, cause: %v", lw.rk, path, lw.zkAddr(), err)
		logger.Errorf(errStr)
		return nil, bizerror.Wrap(err, bizerror.ZKError, errStr)
	}
	res := lw.toUpsertResourceFunc(lw.mesh(), path, string(nodeData))
	if res != nil {
		resList = append(resList, res)
	}
	children, _, err := lw.conn.Children(path)
	if err != nil {
		errStr := fmt.Sprintf("list %s from zk failed, path: %s, addr: %s, cause: %v", lw.rk, path, lw.zkAddr(), err)
		logger.Errorf(errStr)
		return nil, bizerror.Wrap(err, bizerror.ZKError, errStr)
	}
	if len(children) == 0 {
		return resList, nil
	}
	for _, childPath := range children {
		resources, err := lw.listRecursively(path + constants.PathSeparator + childPath)
		if err != nil {
			return nil, err
		}
		resList = append(resList, resources...)
	}
	return resList, nil
}

func (lw *ListerWatcher[T]) Watch(_ metav1.ListOptions) (watch.Interface, error) {
	watcher := zkwatcher.NewRecursiveWatcher(lw.conn, lw.basePath)
	go func() {
		for {
			select {
			case event, ok := <-watcher.EventChan():
				if !ok {
					logger.Warnf("zookeeper watcher stopped, path: %s, addr: %s", lw.basePath, lw.zkAddr())
					return
				}
				lw.handleEvent(event)
			case <-lw.stopChan:
				logger.Warnf("stop watching %s in %s,", lw.basePath, lw.zkAddr())
				return
			}

		}
	}()
	err := watcher.StartAsync()
	lw.watcher = watcher
	if err != nil {
		lw.Stop()
		return nil, err
	}
	return lw, nil
}

func (lw *ListerWatcher[T]) handleEvent(event zkwatcher.ZookeeperEvent) {
	switch event.Type {
	case zkwatcher.NodeCreated:
		res := lw.toUpsertResourceFunc(lw.mesh(), event.Path, event.Data)
		if res == nil {
			logger.Warnf("skip creating resource, parse zookeeper node data to %s failed, path: %s, zk: %s, event: %v",
				lw.rk, event.Path, lw.zkAddr(), event)
			return
		}
		logger.Infof("%s added, rsKey: %s", lw.rk, res.ResourceKey())
		lw.resultChan <- watch.Event{
			Type:   watch.Added,
			Object: res,
		}
	case zkwatcher.NodeChanged:
		res := lw.toUpsertResourceFunc(lw.mesh(), event.Path, event.Data)
		if res == nil {
			logger.Warnf("skip updating resource, parse zookeeper node data to %s failed, path: %s, zk: %s, event: %v",
				lw.rk, event.Path, lw.zkAddr(), event)
			return
		}
		logger.Infof("%s modified, rsKey: %s", lw.rk, res.ResourceKey())
		lw.resultChan <- watch.Event{
			Type:   watch.Modified,
			Object: res,
		}
	case zkwatcher.NodeDeleted:
		res := lw.toDeleteResourceFunc(lw.mesh(), event.Path)
		if res == nil {
			logger.Warnf("skip deleting resource, parse zookeeper node data to %s failed, path: %s, zk: %s, event: %v",
				lw.rk, event.Path, lw.zkAddr(), event)
			return
		}
		logger.Infof("%s deleted, rsKey: %s", lw.rk, res.ResourceKey())
		lw.resultChan <- watch.Event{
			Type:   watch.Deleted,
			Object: res,
		}
	default:
		logger.Warnf("unknown event type, event: %v", event)
	}
}

func (lw *ListerWatcher[T]) zkAddr() string {
	if lw.zkAddress != "" {
		return lw.zkAddress
	}
	return lw.cfg.Address.Registry
}

func (lw *ListerWatcher[T]) mesh() string {
	return lw.cfg.ID
}

func (lw *ListerWatcher[T]) ResourceKind() coremodel.ResourceKind {
	return lw.rk
}

func (lw *ListerWatcher[T]) TransformFunc() cache.TransformFunc {
	return nil
}

func (lw *ListerWatcher[T]) Stop() {
	if lw.watcher != nil {
		lw.watcher.Stop()
	}
	close(lw.stopChan)
	close(lw.resultChan)
	lw.conn.Close()
}

func (lw *ListerWatcher[T]) ResultChan() <-chan watch.Event {
	return lw.resultChan
}
