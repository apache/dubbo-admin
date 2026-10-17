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
	"reflect"
	"strings"

	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

type ZKMetadataEventSubscriber struct {
	emitter     events.Emitter
	storeRouter store.Router
}

func NewZKMetadataEventSubscriber(eventEmitter events.Emitter, storeRouter store.Router) *ZKMetadataEventSubscriber {
	return &ZKMetadataEventSubscriber{
		emitter:     eventEmitter,
		storeRouter: storeRouter,
	}
}

func (z *ZKMetadataEventSubscriber) ResourceKind() coremodel.ResourceKind {
	return meshresource.ZKMetadataKind
}

func (z *ZKMetadataEventSubscriber) Name() string {
	return "Discovery-" + z.ResourceKind().ToString()
}

func (z *ZKMetadataEventSubscriber) AsyncEnabled() bool {
	return true
}

func (z *ZKMetadataEventSubscriber) ProcessEvent(event events.Event) error {
	newObj, ok := event.NewObj().(*meshresource.ZKMetadataResource)
	if !ok && event.NewObj() != nil {
		return bizerror.NewAssertionError(reflect.TypeOf(newObj), event.NewObj())
	}
	oldObj, ok := event.OldObj().(*meshresource.ZKMetadataResource)
	if !ok && event.OldObj() != nil {
		return bizerror.NewAssertionError(reflect.TypeOf(oldObj), event.OldObj())
	}
	var processErr error
	switch event.Type() {
	case cache.Added, cache.Updated, cache.Replaced, cache.Sync:
		if newObj == nil || newObj.Spec == nil {
			errStr := "process zk metadata upsert event, but new obj is nil, skipped processing"
			logger.Errorf(errStr)
			return bizerror.New(bizerror.EventError, errStr)
		}
		processErr = z.processUpsert(newObj)
	case cache.Deleted:
		if oldObj == nil {
			errStr := "process zk metadata delete event, but old obj is nil, skipped processing"
			logger.Errorf(errStr)
			return bizerror.New(bizerror.EventError, errStr)
		}
		// Metadata is an ephemeral znode, dubbo client only adds/updates the metadata znode and never deletes.
		// And we can't identify the service only by the node path, so for delete event, we just ignored
		logger.Infof("ignored zk metadata delete event")
	}
	if processErr != nil {
		logger.Errorf("process zk metadata event failed, cause: %s, event: %s", processErr.Error(), event.String())
		return processErr
	}
	logger.Infof("process zk metadata event successfully, event: %s", event.String())
	return nil
}

func (z *ZKMetadataEventSubscriber) processUpsert(metadataRes *meshresource.ZKMetadataResource) error {
	paths := strings.Split(metadataRes.Spec.NodePath, constants.PathSeparator)
	if len(paths) < 2 {
		return bizerror.New(bizerror.ZKError, fmt.Sprintf("invalid zk metadata node path: %s", metadataRes.Spec.NodePath))
	}
	if paths[len(paths)-2] == constants.ProviderSide {
		return processMetadataUpsert[*meshresource.ServiceProviderMetadataResource](
			metadataRes, meshresource.ToServiceProviderMetadataRes, z.storeRouter, z.emitter)
	} else if paths[len(paths)-2] == constants.ConsumerSide {
		return processMetadataUpsert[*meshresource.ServiceConsumerMetadataResource](
			metadataRes, meshresource.ToServiceConsumerMetadataByRawData, z.storeRouter, z.emitter)
	}
	logger.Warnf("unknown metadata, node path: %s, node data: %s", metadataRes.Spec.NodePath, metadataRes.Spec.NodeData)
	return nil
}

// processMetadataUpsert handle service provider/consumer metadata upsert
func processMetadataUpsert[T coremodel.Resource](
	zkMetadataRes *meshresource.ZKMetadataResource,
	toMetadataRes meshresource.ToMetadataResFunc,
	router store.Router,
	emitter events.Emitter) error {
	newMetadataRes := toMetadataRes(zkMetadataRes.Mesh, zkMetadataRes.Spec.NodeData)
	if newMetadataRes == nil {
		logger.Errorf("cannot unmarshal metadata in zk %s, raw content: %s", zkMetadataRes.Mesh, zkMetadataRes.Spec.NodeData)
		return bizerror.New(bizerror.ZKError, "cannot unmarshal metadata")
	}
	st, err := router.ResourceKindRoute(newMetadataRes.ResourceKind())
	if err != nil {
		logger.Errorf("get %s store failed, cause: %s", newMetadataRes.ResourceKind(), err.Error())
		return err
	}
	oldRes, exists, err := st.GetByKey(newMetadataRes.ResourceKey())
	if err != nil {
		logger.Errorf("get metadata %s from store failed, cause: %s", newMetadataRes.ResourceKey(), err.Error())
		return err
	}
	if !exists {
		err := st.Add(newMetadataRes)
		if err != nil {
			logger.Errorf("add metadata %s to store failed, cause: %s", newMetadataRes.ResourceKey(), err.Error())
			return err
		}
		recordMetadataPlatformEvent(router, newMetadataRes, "added")
		emitter.Send(events.NewResourceChangedEvent(cache.Added, nil, newMetadataRes))
		return nil
	}

	err = st.Update(newMetadataRes)
	if err != nil {
		logger.Errorf("update metadata %s to store failed, cause: %s", newMetadataRes.ResourceKey(), err.Error())
		return err
	}

	var oldMetadataRes T
	oldMetadataRes, ok := oldRes.(T)
	if !ok {
		logger.Errorf("cannot convert old metadata %s to %T", newMetadataRes.ResourceKey(), oldMetadataRes)
		return bizerror.NewAssertionError(reflect.TypeOf(oldMetadataRes), oldRes)
	}

	recordMetadataPlatformEvent(router, newMetadataRes, "updated")
	emitter.Send(events.NewResourceChangedEvent(cache.Updated, oldMetadataRes, newMetadataRes))
	return nil
}

func recordMetadataPlatformEvent(router store.Router, res coremodel.Resource, action string) {
	switch item := res.(type) {
	case *meshresource.ServiceProviderMetadataResource:
		if item.Spec == nil {
			return
		}
		RecordRegistryEvent(router, RegistryEventInput{
			Mesh:        item.Mesh,
			Source:      "Zookeeper",
			SourceType:  "zookeeper",
			Category:    "metadata",
			Action:      action,
			Message:     fmt.Sprintf("Zookeeper provider metadata %s: %s -> %s", action, item.Spec.ProviderAppName, item.Spec.ServiceName),
			AppName:     item.Spec.ProviderAppName,
			ServiceName: item.Spec.ServiceName,
		})
	case *meshresource.ServiceConsumerMetadataResource:
		if item.Spec == nil {
			return
		}
		RecordRegistryEvent(router, RegistryEventInput{
			Mesh:        item.Mesh,
			Source:      "Zookeeper",
			SourceType:  "zookeeper",
			Category:    "metadata",
			Action:      action,
			Message:     fmt.Sprintf("Zookeeper consumer metadata %s: %s -> %s", action, item.Spec.ConsumerAppName, item.Spec.ServiceName),
			AppName:     item.Spec.ConsumerAppName,
			ServiceName: item.Spec.ServiceName,
		})
	}
}
