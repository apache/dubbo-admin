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
	"k8s.io/client-go/tools/cache"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

type InstanceEventSubscriber struct {
	appStore      store.ResourceStore
	instanceStore store.ResourceStore
	emitter       events.Emitter
}

func NewInstanceEventSubscriber(
	appStore store.ResourceStore,
	instanceStore store.ResourceStore,
	emitter events.Emitter) *InstanceEventSubscriber {
	return &InstanceEventSubscriber{
		appStore:      appStore,
		instanceStore: instanceStore,
		emitter:       emitter,
	}
}

func (s *InstanceEventSubscriber) ResourceKind() coremodel.ResourceKind {
	return meshresource.InstanceKind
}

func (s *InstanceEventSubscriber) Name() string {
	return "Discovery-" + s.ResourceKind().ToString()
}

func (s *InstanceEventSubscriber) AsyncEnabled() bool {
	return true
}

func (s *InstanceEventSubscriber) ProcessEvent(event events.Event) error {
	newObj, ok := event.NewObj().(*meshresource.InstanceResource)
	if !ok && event.NewObj() != nil {
		return bizerror.NewAssertionError(meshresource.InstanceKind, event.NewObj())
	}
	oldObj, ok := event.OldObj().(*meshresource.InstanceResource)
	if !ok && event.OldObj() != nil {
		return bizerror.NewAssertionError(meshresource.InstanceKind, event.OldObj())
	}
	var instanceRes *meshresource.InstanceResource
	if newObj != nil {
		instanceRes = newObj
	} else {
		instanceRes = oldObj
	}
	instanceResList, err := s.instanceStore.ListByIndexes([]index.IndexCondition{
		{IndexName: index.ByMeshIndex, Value: instanceRes.Mesh, Operator: index.Equals},
		{IndexName: index.ByInstanceAppNameIndex, Value: instanceRes.Spec.AppName, Operator: index.Equals},
	})

	appResKey := coremodel.BuildResourceKey(instanceRes.Mesh, instanceRes.Spec.AppName)
	res, exists, err := s.appStore.GetByKey(appResKey)
	if err != nil {
		return err
	}
	if !exists {
		appRes := BuildAppResource(instanceRes.Mesh, instanceRes.Spec.AppName, int64(len(instanceResList)))
		err := s.appStore.Add(appRes)
		if err != nil {
			logger.Errorf("add app resource failed, app: %s, err: %s", appResKey, err.Error())
			return err
		}
		s.emitter.Send(events.NewResourceChangedEvent(cache.Added, nil, appRes))
		return nil
	}
	appRes, ok := res.(*meshresource.ApplicationResource)
	if !ok {
		return bizerror.NewAssertionError(meshresource.ApplicationKind, res)
	}
	appRes.Spec.InstanceCount = int64(len(instanceResList))
	err = s.appStore.Update(appRes)
	if err != nil {
		logger.Errorf("update app resource failed, app: %s, err: %s", appResKey, err.Error())
		return err
	}
	return nil
}

func BuildAppResource(mesh string, name string, instanceCount int64) *meshresource.ApplicationResource {
	appRes := meshresource.NewApplicationResourceWithAttributes(name, mesh)
	appRes.Spec = &meshproto.Application{
		Name:          name,
		InstanceCount: instanceCount,
	}
	return appRes
}
