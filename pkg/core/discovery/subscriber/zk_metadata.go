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
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"k8s.io/client-go/tools/cache"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
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
		processErr = z.processApplicationMetadataDelete(oldObj)
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
	if len(paths) == 5 && paths[1] == "dubbo" && paths[3] == "providers" {
		resource := zkInterfaceRegistration(metadataRes.Mesh, metadataRes.Spec.NodePath)
		if resource == nil {
			return fmt.Errorf("invalid zk provider registration %q", metadataRes.Spec.NodePath)
		}
		return z.upsertApplicationMetadata(resource)
	}
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
	if app, revision, ok := applicationMetadataPath(metadataRes.Spec.NodePath); ok {
		resource, err := meshresource.ParseRPCInstanceMetadata(metadataRes.Mesh, app, revision, metadataRes.Spec.NodeData)
		if err != nil {
			return err
		}
		return z.upsertApplicationMetadata(resource)
	}
	logger.Warnf("unknown metadata, node path: %s, node data: %s", metadataRes.Spec.NodePath, metadataRes.Spec.NodeData)
	return nil
}

func applicationMetadataPath(path string) (string, string, bool) {
	parts := strings.Split(path, constants.PathSeparator)
	if len(parts) != 5 || parts[1] != "dubbo" || parts[2] != "metadata" ||
		parts[3] == "" || parts[4] == "" {
		return "", "", false
	}
	return parts[3], parts[4], true
}

func (z *ZKMetadataEventSubscriber) upsertApplicationMetadata(resource *meshresource.RPCInstanceMetadataResource) error {
	st, err := z.storeRouter.ResourceKindRoute(meshresource.RPCInstanceMetadataKind)
	if err != nil {
		return err
	}
	old, exists, err := st.GetByKey(resource.ResourceKey())
	if err != nil {
		return err
	}
	if !exists {
		if err := st.Add(resource); err != nil {
			return err
		}
		z.emitter.Send(events.NewResourceChangedEvent(cache.Added, nil, resource))
		return nil
	}
	if err := st.Update(resource); err != nil {
		return err
	}
	z.emitter.Send(events.NewResourceChangedEvent(cache.Updated, old.(coremodel.Resource), resource))
	return nil
}

func (z *ZKMetadataEventSubscriber) processApplicationMetadataDelete(resource *meshresource.ZKMetadataResource) error {
	if resource.Spec == nil {
		return nil
	}
	key := zkInterfaceRegistration(resource.Mesh, resource.Spec.NodePath)
	if key == nil {
		app, revision, ok := applicationMetadataPath(resource.Spec.NodePath)
		if !ok {
			return nil
		}
		key = meshresource.NewRPCInstanceMetadataResourceWithAttributes(app+"@"+revision, resource.Mesh)
	}
	st, err := z.storeRouter.ResourceKindRoute(meshresource.RPCInstanceMetadataKind)
	if err != nil {
		return err
	}
	old, exists, err := st.GetByKey(key.ResourceKey())
	if err != nil || !exists {
		return err
	}
	if err := st.Delete(old); err != nil {
		return err
	}
	z.emitter.Send(events.NewResourceChangedEvent(cache.Deleted, old.(coremodel.Resource), nil))
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

func zkInterfaceRegistration(mesh, path string) *meshresource.RPCInstanceMetadataResource {
	parts := strings.Split(path, constants.PathSeparator)
	if len(parts) != 5 || parts[1] != "dubbo" || parts[3] != "providers" {
		return nil
	}
	decoded, err := url.PathUnescape(parts[4])
	if err != nil {
		return nil
	}
	endpoint, err := url.Parse(decoded)
	if err != nil || endpoint.Hostname() == "" || endpoint.Scheme == "" {
		return nil
	}
	port, err := strconv.ParseInt(endpoint.Port(), 10, 64)
	if err != nil || port <= 0 {
		return nil
	}
	app := endpoint.Query().Get(constants.Application)
	service := parts[2]
	if service == "" {
		return nil
	}
	group, version := endpoint.Query().Get(constants.GroupKey), endpoint.Query().Get(constants.VersionKey)
	available := endpoint.Query().Get("disabled") != "true" && endpoint.Query().Get("enabled") != "false"
	if weight := endpoint.Query().Get("weight"); weight != "" {
		value, err := strconv.ParseFloat(weight, 64)
		if err != nil {
			return nil
		}
		available = available && value > 0
	}
	name := fmt.Sprintf("interface:%s:%s:%s:%d:%s:%s:%s",
		service, app, endpoint.Hostname(), port, group, version, endpoint.Scheme)
	res := meshresource.NewRPCInstanceMetadataResourceWithAttributes(name, mesh)
	res.Spec = &meshproto.RPCInstanceMetadata{
		App:          app,
		InstanceName: meshresource.BuildInstanceResName(app, endpoint.Hostname(), port),
		Available:    &available,
		Services: map[string]*meshproto.ServiceInfo{
			meshresource.BuildServiceIdentityKey(service, version, group): {
				Name: service, Group: group, Version: version, Protocol: endpoint.Scheme, Port: port,
			},
		},
	}
	return res
}
