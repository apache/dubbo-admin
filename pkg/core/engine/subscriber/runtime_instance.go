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
	"errors"
	"reflect"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	enginecfg "github.com/apache/dubbo-admin/pkg/config/engine"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

type RuntimeInstanceEventSubscriber struct {
	instanceStore store.ResourceStore
	eventEmitter  events.Emitter
}

func NewRuntimeInstanceEventSubscriber(instanceResourceStore store.ResourceStore, emitter events.Emitter) events.Subscriber {
	return &RuntimeInstanceEventSubscriber{
		instanceStore: instanceResourceStore,
		eventEmitter:  emitter,
	}
}

func (s *RuntimeInstanceEventSubscriber) ResourceKind() coremodel.ResourceKind {
	return meshresource.RuntimeInstanceKind
}

func (s *RuntimeInstanceEventSubscriber) Name() string {
	return "Engine-" + s.ResourceKind().ToString()
}

func (s *RuntimeInstanceEventSubscriber) ProcessEvent(event events.Event) error {
	newObj, ok := event.NewObj().(*meshresource.RuntimeInstanceResource)
	if !ok && event.NewObj() != nil {
		return bizerror.NewAssertionError(meshresource.RuntimeInstanceKind, reflect.TypeOf(event.NewObj()).Name())
	}
	oldObj, ok := event.OldObj().(*meshresource.RuntimeInstanceResource)
	if !ok && event.OldObj() != nil {
		return bizerror.NewAssertionError(meshresource.RuntimeInstanceKind, reflect.TypeOf(event.OldObj()).Name())
	}
	var processErr error
	switch event.Type() {
	case cache.Added, cache.Updated, cache.Replaced, cache.Sync:
		if newObj == nil {
			errStr := "process runtime instance upsert event, but new obj is nil, skipped processing"
			logger.Error(errStr)
			return errors.New(errStr)
		}
		processErr = s.processUpsert(newObj)
	case cache.Deleted:
		if oldObj == nil {
			errStr := "process runtime instance delete event, but old obj is nil, skipped processing"
			logger.Error(errStr)
			return errors.New(errStr)
		}
		processErr = s.processDelete(oldObj)
	}
	if processErr != nil {
		logger.Errorf("process runtime instance event failed, cause: %s event: %s", processErr.Error(), event.String())
		return processErr
	}
	logger.Infof("process runtime instance event successfully, event: %s", event.String())
	return nil
}

// processUpsert when runtime instance added or updated, we should add/update the corresponding instance resource
func (s *RuntimeInstanceEventSubscriber) processUpsert(rtInstanceRes *meshresource.RuntimeInstanceResource) error {
	var instanceResource *meshresource.InstanceResource
	var err error
	switch rtInstanceRes.Spec.SourceEngineType {
	case string(enginecfg.Kubernetes):
		instanceResource, err = s.getRelatedInstanceByIP(rtInstanceRes)
	default:
		instanceResource, err = s.getRelatedInstanceByName(rtInstanceRes)
	}
	if err != nil {
		return err
	}
	// if instance resource exists, that is to say the rpc instance exists in remote registry and has been watched by discovery.
	// so we should merge the runtime info into it
	if instanceResource != nil {
		meshresource.MergeRuntimeInstanceIntoInstance(rtInstanceRes, instanceResource)
		return s.instanceStore.Update(instanceResource)
	}
	// if instance resource does not exist, that is to say the rpc instance does not exist in remote registry.
	// we need to check whether the runtime instance resource is enough to create a new instance resource
	if !checkAttributesEnough(rtInstanceRes) {
		logger.Warnf("cannot identify runtime instance %s to a dubbo instance, skipped creating new instance", rtInstanceRes.ResourceKey())
		return nil
	}
	// if conditions met, we should create a new instance resource by runtime instance
	instanceRes := meshresource.FromRuntimeInstance(rtInstanceRes)
	if err = s.instanceStore.Add(instanceRes); err != nil {
		logger.Errorf("add instance resource failed, instance: %s, err: %s", instanceRes.ResourceKey(), err.Error())
		return err
	}
	instanceAddEvent := events.NewResourceChangedEvent(cache.Added, nil, instanceRes)
	s.eventEmitter.Send(instanceAddEvent)
	logger.Debugf("runtime instance upsert trigger instance add event, event: %s", instanceAddEvent.String())
	return nil
}

// processDelete when runtime instance deleted, we should delete the corresponding instance resource
func (s *RuntimeInstanceEventSubscriber) processDelete(rtInstanceRes *meshresource.RuntimeInstanceResource) error {
	var instanceResource *meshresource.InstanceResource
	var err error
	switch rtInstanceRes.Spec.SourceEngineType {
	case string(enginecfg.Kubernetes):
		instanceResource, err = s.getRelatedInstanceByIP(rtInstanceRes)
	default:
		instanceResource, err = s.getRelatedInstanceByName(rtInstanceRes)
	}
	if err != nil {
		return err
	}
	if instanceResource == nil {
		logger.Warnf("cannot find instance resource by runtime instance %s, skipped deleting instance", rtInstanceRes.ResourceKey())
		return nil
	}
	if err = s.instanceStore.Delete(instanceResource); err != nil {
		logger.Errorf("delete instance resource failed, instance: %s, err: %s", instanceResource.ResourceKey(), err.Error())
		return err
	}
	instanceDeleteEvent := events.NewResourceChangedEvent(cache.Deleted, instanceResource, nil)
	s.eventEmitter.Send(instanceDeleteEvent)
	logger.Debugf("runtime instance delete trigger instance delete event, event: %s", instanceDeleteEvent.String())
	return nil
}

func (s *RuntimeInstanceEventSubscriber) getRelatedInstanceByName(
	rtInstanceRes *meshresource.RuntimeInstanceResource) (*meshresource.InstanceResource, error) {
	if rtInstanceRes.Spec == nil || strutil.IsBlank(rtInstanceRes.Spec.AppName) ||
		strutil.IsBlank(rtInstanceRes.Spec.Ip) || rtInstanceRes.Spec.RpcPort <= 0 {
		return nil, nil
	}
	instanceResName := meshresource.BuildInstanceResName(rtInstanceRes.Spec.AppName, rtInstanceRes.Spec.Ip, rtInstanceRes.Spec.RpcPort)
	resources, err := s.instanceStore.ListByIndexes([]index.IndexCondition{
		{IndexName: index.ByInstanceNameIndex, Value: instanceResName, Operator: index.Equals},
	})
	if err != nil {
		return nil, err
	}
	if len(resources) == 0 {
		return nil, nil
	}
	instanceResList := make([]*meshresource.InstanceResource, len(resources))
	for i, item := range resources {
		res, ok := item.(*meshresource.InstanceResource)
		if !ok {
			return nil, bizerror.NewAssertionError("InstanceResource", reflect.TypeOf(item).Name())
		}
		instanceResList[i] = res
	}
	if len(instanceResList) > 1 {
		resKeys := slice.Map(instanceResList, func(index int, item *meshresource.InstanceResource) string {
			return item.ResourceKey()
		})
		logger.Warnf("there are more than two instances which have the same name, instance keys: %s, name: %s", resKeys, instanceResName)
		return nil, nil
	}
	return instanceResList[0], nil
}

func (s *RuntimeInstanceEventSubscriber) getRelatedInstanceByIP(
	rtInstanceRes *meshresource.RuntimeInstanceResource) (*meshresource.InstanceResource, error) {
	resources, err := s.instanceStore.ListByIndexes([]index.IndexCondition{
		{IndexName: index.ByInstanceIpIndex, Value: rtInstanceRes.Spec.Ip, Operator: index.Equals},
	})
	if err != nil {
		return nil, err
	}
	if len(resources) == 0 {
		return nil, nil
	}
	instanceResList := make([]*meshresource.InstanceResource, len(resources))
	for i, item := range resources {
		res, ok := item.(*meshresource.InstanceResource)
		if !ok {
			return nil, bizerror.NewAssertionError("InstanceResource", reflect.TypeOf(item).Name())
		}
		instanceResList[i] = res
	}
	if len(instanceResList) > 1 {
		resKeys := slice.Map(instanceResList, func(index int, item *meshresource.InstanceResource) string {
			return item.ResourceKey()
		})
		logger.Warnf("there are instances which have same ip, instance keys: %s, ip: %s", resKeys, rtInstanceRes.Spec.Ip)
		return nil, nil
	}
	return instanceResList[0], nil
}

func checkAttributesEnough(rtInstanceRes *meshresource.RuntimeInstanceResource) bool {
	if rtInstanceRes.Spec == nil || strutil.IsBlank(rtInstanceRes.Spec.AppName) ||
		strutil.IsBlank(rtInstanceRes.Spec.Ip) || rtInstanceRes.Spec.RpcPort <= 0 ||
		strutil.IsBlank(rtInstanceRes.Mesh) || constants.DefaultMesh == rtInstanceRes.Mesh {
		return false
	}
	return true
}
