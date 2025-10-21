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

	"github.com/duke-git/lancet/v2/strutil"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

type RuntimeInstanceEventSubscriber struct {
	instanceResourceStore store.ResourceStore
	eventEmitter          events.Emitter
}

func (s *RuntimeInstanceEventSubscriber) ResourceKind() coremodel.ResourceKind {
	return meshresource.RuntimeInstanceKind
}

func (s *RuntimeInstanceEventSubscriber) Name() string {
	return "Engine-" + s.ResourceKind().ToString()
}

func (s *RuntimeInstanceEventSubscriber) ProcessEvent(event events.Event) error {
	newObj, ok := event.NewObj().(*meshresource.RuntimeInstanceResource)
	if !ok && newObj != nil {
		return bizerror.NewAssertionError(meshresource.RuntimeInstanceKind, reflect.TypeOf(event.NewObj()).Name())
	}
	oldObj, ok := event.OldObj().(*meshresource.RuntimeInstanceResource)
	if !ok && oldObj != nil {
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
	eventStr := event.String()
	if processErr == nil {
		logger.Infof("process runtime instance event successfully, event: %s", eventStr)
	} else {
		logger.Errorf("process runtime instance event failed, event: %s, err: %s", eventStr, processErr.Error())
	}
	return processErr
}

func (s *RuntimeInstanceEventSubscriber) getRelatedInstanceResource(
	rtInstance *meshresource.RuntimeInstanceResource) (*meshresource.InstanceResource, error) {
	resources, err := s.instanceResourceStore.ListByIndexes(map[string]string{
		index.ByInstanceIpIndex: rtInstance.Spec.Ip,
	})
	if err != nil {
		return nil, err
	}
	if len(resources) == 0 {
		return nil, nil
	}
	instanceResources := make([]*meshresource.InstanceResource, len(resources))
	for i, item := range resources {
		if res, ok := item.(*meshresource.InstanceResource); ok {
			instanceResources[i] = res
		} else {
			return nil, bizerror.NewAssertionError("InstanceResource", reflect.TypeOf(item).Name())
		}
	}
	return instanceResources[0], nil
}

func (s *RuntimeInstanceEventSubscriber) mergeRuntimeInstance(
	instanceRes *meshresource.InstanceResource,
	rtInstanceRes *meshresource.RuntimeInstanceResource) {
	instanceRes.Name = rtInstanceRes.Name
	instanceRes.Spec.Name = rtInstanceRes.Spec.Name
	instanceRes.Spec.Ip = rtInstanceRes.Spec.Ip
	instanceRes.Labels = rtInstanceRes.Labels
	instanceRes.Spec.Image = rtInstanceRes.Spec.Image
	instanceRes.Spec.CreateTime = rtInstanceRes.Spec.CreateTime
	instanceRes.Spec.StartTime = rtInstanceRes.Spec.StartTime
	instanceRes.Spec.ReadyTime = rtInstanceRes.Spec.ReadyTime
	instanceRes.Spec.DeployState = rtInstanceRes.Spec.Phase
	instanceRes.Spec.WorkloadType = rtInstanceRes.Spec.WorkloadType
	instanceRes.Spec.WorkloadName = rtInstanceRes.Spec.WorkloadName
	instanceRes.Spec.Node = rtInstanceRes.Spec.Node
	instanceRes.Spec.Probes = rtInstanceRes.Spec.Probes
	instanceRes.Spec.Conditions = rtInstanceRes.Spec.Conditions
}

func (s *RuntimeInstanceEventSubscriber) fromRuntimeInstance(
	rtInstanceRes *meshresource.RuntimeInstanceResource) *meshresource.InstanceResource {
	instanceRes := meshresource.NewInstanceResourceWithAttributes(rtInstanceRes.Name, rtInstanceRes.Mesh)
	s.mergeRuntimeInstance(instanceRes, rtInstanceRes)
	return instanceRes
}

// processUpsert when runtime instance added or updated, we should add/update the corresponding instance resource
func (s *RuntimeInstanceEventSubscriber) processUpsert(rtInstanceRes *meshresource.RuntimeInstanceResource) error {
	instanceResource, err := s.getRelatedInstanceResource(rtInstanceRes)
	if err != nil {
		return err
	}
	// If instance resource exists, the rpc instance resource exists in remote registry and has been watched by discovery.
	// So we should merge the runtime info into it
	if instanceResource != nil {
		s.mergeRuntimeInstance(instanceResource, rtInstanceRes)
		return s.instanceResourceStore.Update(instanceResource)
	}
	// If instance resource does not exist, we should create a new instance resource by runtime instance
	// If the app name is empty, we cannot identify it as a dubbo app, so we skip it
	if strutil.IsBlank(rtInstanceRes.Spec.AppName) {
		logger.Warnf("cannot identify runtime instance %s as a dubbo app, skipped updating instance", rtInstanceRes.Name)
		return nil
	}
	// Otherwise we can create a new instance resource by runtime instance
	instanceRes := s.fromRuntimeInstance(rtInstanceRes)
	if err = s.instanceResourceStore.Add(instanceRes); err != nil {
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
	instanceResource, err := s.getRelatedInstanceResource(rtInstanceRes)
	if err != nil {
		return err
	}
	if instanceResource == nil {
		return nil
	}
	if err = s.instanceResourceStore.Delete(instanceResource.ResourceKey()); err != nil {
		logger.Errorf("delete instance resource failed, instance: %s, err: %s", instanceResource.ResourceKey(), err.Error())
		return err
	}
	instanceDeleteEvent := events.NewResourceChangedEvent(cache.Deleted, instanceResource, nil)
	s.eventEmitter.Send(instanceDeleteEvent)
	logger.Debugf("runtime instance delete trigger instance delete event, event: %s", instanceDeleteEvent.String())
	return nil
}

func NewRuntimeInstanceEventSubscriber(instanceResourceStore store.ResourceStore, emitter events.Emitter) events.Subscriber {
	return &RuntimeInstanceEventSubscriber{
		instanceResourceStore: instanceResourceStore,
		eventEmitter:          emitter,
	}
}
